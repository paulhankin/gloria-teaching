package pipeline

import (
	"fmt"
	"path/filepath"
	"strings"

	"learningmaterial/internal/store"
)

// This file imports the commits the agent produced in the sandbox clones
// back into the live repositories. It replaces the worktree-mode merge() in
// sandbox mode: the request branch only exists inside the disposable clones,
// so the live repositories fetch exactly the recorded branch of each clone
// into a temporary ref, validate it against the recorded base commits, and
// rebase it onto the current main. Live branches are never force-updated.
//
// Failure atomicity across the two repositories is a two-phase commit:
// phase A fetches and validates both clones, phase B rebases both fetched
// tips onto their live mains, and only when both rebases succeeded are the
// two mains fast-forwarded. A conflict in either repository therefore leaves
// both live mains untouched.
//
// Crash recovery is per repository: after each successful fast-forward the
// rebased tip is recorded in the sandbox metadata (ImportedCoreCommit /
// ImportedWorksheetCommit), so a restart between the two fast-forwards (or
// between merge and push) can redo only the missing side or skip straight to
// push and rebuild.

// publishSandbox is the sandbox-mode publication step of publish(): it
// imports the clone commits of one request into the live repositories,
// unless a previous attempt already did. The caller must hold publishMu.
func (p *Pipeline) publishSandbox(it store.Request, username string) error {
	paths, err := sandboxPathsFor(p.cfg.SandboxRoot, it.ID, username)
	if err != nil {
		return err
	}
	meta, err := readMetadata(paths)
	if err != nil {
		return fmt.Errorf("publish request #%d: %w", it.ID, err)
	}
	applied, err := p.importsApplied(meta)
	if err != nil {
		return fmt.Errorf("check the imported commits of request #%d: %w", it.ID, err)
	}
	if applied {
		p.Logf("#%d: the imported commits are already on both live mains; continuing with the push", it.ID)
		return nil
	}
	meta.Phase = phaseImporting
	if err := writeMetadata(paths, meta); err != nil {
		return fmt.Errorf("record the importing phase: %w", err)
	}
	if err := p.importCommits(it, paths, &meta); err != nil {
		return fmt.Errorf("import: %w", err)
	}
	// The commits are imported; the push, the rebuild, and the cleanup
	// follow in publish().
	meta.Phase = phasePublished
	if err := writeMetadata(paths, meta); err != nil {
		p.Logf("#%d: record the published phase: %v", it.ID, err)
	}
	p.Logf("#%d: imported the sandbox commits into the live repositories", it.ID)
	return nil
}

// importRepo pairs one sandbox clone with its live repository.
type importRepo struct {
	name     string // "core" or "worksheet", for error messages
	live     string // live repository checkout (main is checked out)
	clone    string // sandbox clone the agent committed in
	base     string // clone base commit recorded at creation time
	imported string // rebased tip already recorded by a previous attempt

	tmpRef    string // fetch destination inside the live repository
	tmpBranch string // rebase work branch inside the live repository
	hasBranch bool   // tmpBranch exists (set after a successful rebase)
	fetched   string // validated fetched tip (equals the clone tip)
	rebased   string // fetched tip rebased onto the live main
}

// importCommits imports the agent's commits from both sandbox clones into
// the live repositories. It is idempotent per repository: a side whose
// imported tip is already an ancestor of the live main is skipped. The
// caller must hold publishMu.
func (p *Pipeline) importCommits(it store.Request, paths sandboxPaths, meta *sandboxMetadata) error {
	username := meta.Username
	if username == "" {
		var err error
		username, err = p.requestUsername(it)
		if err != nil {
			return err
		}
	}
	if meta.CoreCommit == "" || meta.WorksheetCommit == "" {
		return fmt.Errorf("sandbox metadata of request #%d has no recorded base commits", it.ID)
	}

	repos := []*importRepo{
		{
			name:     "core",
			live:     p.cfg.Repo,
			clone:    paths.Workspace,
			base:     meta.CoreCommit,
			imported: meta.ImportedCoreCommit,
		},
		{
			name:     "worksheet",
			live:     p.userRepo(username),
			clone:    paths.WorkspaceUserRepo,
			base:     meta.WorksheetCommit,
			imported: meta.ImportedWorksheetCommit,
		},
	}

	// Idempotent resume: a repository whose imported tip already reached its
	// live main needs neither fetch nor validation (its clone may already be
	// gone); it is marked done and fast-forwarded again below.
	pending := 0
	for _, r := range repos {
		r.tmpRef = fmt.Sprintf("refs/learningmaterial/req-%d-%s", it.ID, r.name)
		r.tmpBranch = fmt.Sprintf("learningmaterial/req-%d-%s", it.ID, r.name)
		done, err := p.importApplied(r)
		if err != nil {
			return err
		}
		if done {
			r.rebased = r.imported
			p.Logf("#%d: %s repository already contains the imported commits; skipping the import", it.ID, r.name)
			continue
		}
		pending++
	}
	if pending == 0 {
		return p.ffMergedImported(repos)
	}

	// Phase A: fetch the recorded request branch of each pending clone into a
	// temporary ref of its live repository and validate it. Live main is not
	// modified. A fetched ref never escapes this function.
	fetched := make(map[*importRepo]bool)
	defer func() {
		for r := range fetched {
			p.deleteTempRef(r)
		}
	}()
	advanced := 0
	for _, r := range repos {
		if r.rebased != "" {
			continue
		}
		if err := p.fetchAndValidate(it, r, meta, paths, username, fetched); err != nil {
			return err
		}
		if r.fetched != r.base {
			advanced++
		}
	}
	if advanced == 0 {
		return fmt.Errorf("the agent produced no commits: both sandbox clones are still at their base commits")
	}

	// Phase B, step 1: rebase both fetched tips onto their live mains. Only
	// when both rebases succeed may either main move, so a conflict in the
	// worksheet repository cannot strand a half-imported core repository.
	// Each repository ends the step back on its main branch.
	type rebaseState struct {
		repo         *importRepo
		previousHEAD string
	}
	var rebased []rebaseState
	rollback := func() {
		for i := len(rebased) - 1; i >= 0; i-- {
			st := rebased[i]
			if out, err := p.git(st.repo.live, "checkout", "main"); err != nil {
				p.Logf("#%d: rollback %s checkout: %v: %s", it.ID, st.repo.name, err, strings.TrimSpace(out))
				if _, err := p.git(st.repo.live, "checkout", "--detach", st.previousHEAD); err != nil {
					p.Logf("#%d: rollback %s detached checkout: %v", it.ID, st.repo.name, err)
				}
			}
			if out, err := p.git(st.repo.live, "branch", "-D", st.repo.tmpBranch); err != nil {
				p.Logf("#%d: rollback %s branch deletion: %v: %s", it.ID, st.repo.name, err, strings.TrimSpace(out))
			}
		}
		rebased = nil
	}
	for _, r := range repos {
		if r.rebased != "" {
			continue
		}
		if r.fetched == r.base {
			// The clone did not move (allowed when the other one did); its
			// recorded base is an ancestor of the live main the clone was
			// created from, so the final fast-forward is a no-op.
			r.rebased = r.base
			continue
		}
		previousHEAD, err := p.rebaseOnto(r)
		if err != nil {
			rollback()
			return err
		}
		rebased = append(rebased, rebaseState{repo: r, previousHEAD: previousHEAD})
		if _, err := p.git(r.live, "checkout", "main"); err != nil {
			rollback()
			return fmt.Errorf("%s repository: check out main after the rebase: %w", r.name, err)
		}
	}

	// Phase B, step 2: fast-forward both mains. A failure here can only be a
	// race with another writer (publishMu serializes imports), so the temp
	// branches are kept for diagnosis and metadata is left untouched: the
	// next attempt refetches the still-intact clones.
	for _, r := range repos {
		if r.name == "core" {
			meta.ImportedCoreCommit = r.rebased
		} else {
			meta.ImportedWorksheetCommit = r.rebased
		}
		target := r.rebased
		if r.hasBranch {
			// Rebasing happened on the temporary branch this run.
			target = r.tmpBranch
		}
		if _, err := p.git(r.live, "merge", "--ff-only", target); err != nil {
			return fmt.Errorf("%s repository: fast-forward main: %w (temp branch %s kept for diagnosis)", r.name, err, r.tmpBranch)
		}
	}

	// Both mains moved; persist the imported tips side by side so a crash
	// between the two metadata writes (or before the push) resumes cleanly.
	for _, r := range repos {
		if err := writeMetadata(paths, *meta); err != nil {
			return fmt.Errorf("record the imported %s tip: %w", r.name, err)
		}
	}

	for _, r := range repos {
		if r.hasBranch {
			if _, err := p.git(r.live, "branch", "-D", r.tmpBranch); err != nil {
				p.Logf("#%d: delete temp branch %s: %v", it.ID, r.tmpBranch, err)
			}
			r.hasBranch = false
		}
		if fetched[r] {
			p.deleteTempRef(r)
			delete(fetched, r)
		}
	}
	return nil
}

// importApplied reports whether the repository's recorded imported tip is
// already contained in its live main.
func (p *Pipeline) importApplied(r *importRepo) (bool, error) {
	if r.imported == "" {
		return false, nil
	}
	contains, err := p.isAncestor(r.live, r.imported, "main")
	if err != nil {
		return false, fmt.Errorf("%s repository: check the recorded imported tip: %w", r.name, err)
	}
	if !contains {
		return false, fmt.Errorf("%s repository: the recorded imported tip %s is not an ancestor of main "+
			"(main was force-updated or the metadata is stale); refusing to continue", r.name, r.imported)
	}
	return true, nil
}

// fetchAndValidate fetches the recorded request branch of one clone into a
// temporary ref of the live repository and validates the result: the fetched
// tip must equal the clone's HEAD, descend from the recorded base commit,
// and (for the core repository) not touch the nested worksheet repository's
// paths. A clone that did not move at all fetches nothing.
func (p *Pipeline) fetchAndValidate(it store.Request, r *importRepo, meta *sandboxMetadata, paths sandboxPaths, username string, fetchedRefs map[*importRepo]bool) error {
	if err := validateClone(r.clone); err != nil {
		return fmt.Errorf("%s clone: %w", r.name, err)
	}
	root, err := filepath.EvalSymlinks(r.clone)
	if err != nil {
		return fmt.Errorf("%s clone: %w", r.name, err)
	}
	if !pathBelow(paths.Root, root) {
		return fmt.Errorf("%s clone %s escapes the request sandbox %s", r.name, root, paths.Root)
	}

	tip, err := p.headCommit(r.clone)
	if err != nil {
		return fmt.Errorf("%s clone: %w", r.name, err)
	}
	if tip == r.base {
		r.fetched = r.base
		return nil
	}

	// Reject rewritten history before fetching: every new commit must
	// descend from the recorded base.
	descends, err := p.isAncestor(r.clone, r.base, tip)
	if err != nil {
		return fmt.Errorf("%s clone: check ancestry: %w", r.name, err)
	}
	if !descends {
		return fmt.Errorf("%s clone tip %s does not descend from the recorded base %s: "+
			"the request branch was rewritten", r.name, shortTip(tip), shortTip(r.base))
	}

	// Fetch exactly the recorded branch into a temporary ref. Fetching the
	// clone's HEAD or an arbitrary ref name would accept whatever the
	// sandbox left behind; the branch name comes from the trusted metadata.
	// The temp ref name is derived from the request ID, so it cannot
	// collide with user branches.
	spec := fmt.Sprintf("+%s:%s", meta.Branch, r.tmpRef)
	if _, err := p.git(r.live, "fetch", "--no-tags", r.clone, spec); err != nil {
		return fmt.Errorf("fetch the %s clone into the live repository: %w", r.name, err)
	}
	// From here on the temp ref exists and must be deleted on every path.
	if fetchedRefs != nil {
		fetchedRefs[r] = true
	}
	fetched, err := p.git(r.live, "rev-parse", r.tmpRef)
	if err != nil {
		return fmt.Errorf("%s repository: resolve the fetched ref: %w", r.name, err)
	}
	r.fetched = strings.TrimSpace(fetched)
	if r.fetched != tip {
		return fmt.Errorf("%s clone moved during the import: HEAD is %s but the fetched branch is %s",
			r.name, shortTip(tip), shortTip(r.fetched))
	}

	// Re-verify ancestry inside the live repository: validation must depend
	// on what was actually imported, not on what the clone claimed.
	descends, err = p.isAncestor(r.live, r.base, r.fetched)
	if err != nil {
		return fmt.Errorf("%s repository: check the fetched ancestry: %w", r.name, err)
	}
	if !descends {
		return fmt.Errorf("%s repository: the fetched tip %s does not descend from the recorded base %s",
			r.name, shortTip(r.fetched), shortTip(r.base))
	}

	if r.name == "core" {
		// The nested worksheet repository is a separate clone; a core commit
		// touching its domain would smuggle unvalidated content past the
		// worksheet import. The nested clone itself is validated above, so a
		// symlink at generate/<username> cannot redirect this check.
		out, err := p.git(r.live, "diff", "--name-only", r.base, r.fetched)
		if err != nil {
			return fmt.Errorf("core repository: list the imported changes: %w", err)
		}
		prefix := "generate/" + username + "/"
		for _, path := range strings.Split(strings.TrimSpace(out), "\n") {
			if strings.TrimSpace(path) == "" {
				continue
			}
			if strings.HasPrefix(path, prefix) {
				return fmt.Errorf("core clone commits touch %s, which belongs to the worksheet repository", path)
			}
		}
	}
	return nil
}

// rebaseOnto replays the fetched commits onto the live main on a temporary
// branch and returns the live repository's previous HEAD. On a conflict the
// rebase is aborted and the error is clean: no live branch was touched. On
// success the repository is left on the temporary branch; the caller checks
// out main again once every repository has been rebased.
func (p *Pipeline) rebaseOnto(r *importRepo) (previousHEAD string, err error) {
	previousHEAD, err = p.headCommit(r.live)
	if err != nil {
		return "", fmt.Errorf("%s repository: record HEAD: %w", r.name, err)
	}
	if _, err := p.git(r.live, "branch", "-f", r.tmpBranch, r.tmpRef); err != nil {
		return "", fmt.Errorf("%s repository: create the import branch: %w", r.name, err)
	}
	failed := true
	defer func() {
		if failed {
			p.git(r.live, "branch", "-D", r.tmpBranch)
		}
	}()
	if _, err := p.git(r.live, "checkout", r.tmpBranch); err != nil {
		return "", fmt.Errorf("%s repository: check out the import branch: %w", r.name, err)
	}
	out, err := p.git(r.live, "rebase", "--onto", "main", r.base, r.tmpBranch)
	if err != nil {
		abortOut, abortErr := p.git(r.live, "rebase", "--abort")
		if abortErr != nil {
			return "", fmt.Errorf("%s repository: rebase onto main failed (%v) and --abort failed too: %v: %s",
				r.name, err, abortErr, strings.TrimSpace(abortOut))
		}
		if _, coErr := p.git(r.live, "checkout", "--detach", previousHEAD); coErr != nil {
			p.Logf("restore the %s checkout after a failed rebase: %v", r.name, coErr)
		}
		return "", fmt.Errorf("the %s changes conflict with the current main (a newer request may have "+
			"touched the same files); no live branch was modified, retry the request to rebase them: %s",
			r.name, tail(out, 400))
	}
	tip, err := p.headCommit(r.live)
	if err != nil {
		return "", fmt.Errorf("%s repository: resolve the rebased tip: %w", r.name, err)
	}
	r.rebased = tip
	r.hasBranch = true
	failed = false
	return previousHEAD, nil
}

// ffMergedImported re-applies the fast-forward of main to the recorded
// imported tip of every repository. After a successful import this is a
// no-op; after a crash between the import and the push it re-fast-forwards a
// repository whose main somehow lost the recorded tip.
func (p *Pipeline) ffMergedImported(repos []*importRepo) error {
	for _, r := range repos {
		if r.rebased == "" {
			return fmt.Errorf("internal error: %s repository has no imported tip to publish", r.name)
		}
		if _, err := p.git(r.live, "merge", "--ff-only", r.rebased); err != nil {
			return fmt.Errorf("%s repository: fast-forward main to the recorded imported tip: %w", r.name, err)
		}
	}
	return nil
}

// deleteTempRef removes the temporary fetch ref of one repository. Deletion
// is best-effort: a leftover temp ref is harmless but never intentional.
func (p *Pipeline) deleteTempRef(r *importRepo) {
	if _, err := p.git(r.live, "update-ref", "-d", r.tmpRef); err != nil {
		p.Logf("delete temp ref %s in the %s repository: %v", r.tmpRef, r.name, err)
	}
}

// importsApplied is the sandbox-mode equivalent of branchesMerged: it
// reports whether the recorded imported tips of both repositories are
// already contained in the live mains. A missing metadata file or missing
// imported tips simply means publication has not happened yet.
func (p *Pipeline) importsApplied(meta sandboxMetadata) (bool, error) {
	if meta.ImportedCoreCommit == "" || meta.ImportedWorksheetCommit == "" {
		return false, nil
	}
	pairs := []struct {
		repo string
		tip  string
	}{
		{p.cfg.Repo, meta.ImportedCoreCommit},
		{p.userRepo(meta.Username), meta.ImportedWorksheetCommit},
	}
	for _, pair := range pairs {
		contains, err := p.isAncestor(pair.repo, pair.tip, "main")
		if err != nil || !contains {
			return contains, err
		}
	}
	return true, nil
}

func shortTip(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}
