package pipeline

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"learningmaterial/internal/sandbox"
	"learningmaterial/internal/store"
)

// setupImportFixture creates a live core repo, a live user repo, their bare
// push remotes and the sandbox clones of request 7, and returns everything
// an import test needs. The pipeline runs in sandbox mode with pushes to the
// local bare remotes.
func setupImportFixture(t *testing.T) (*Pipeline, store.Request, sandboxPaths, sandboxMetadata, string, string) {
	t.Helper()
	p, paths, coreRepo, _, _ := setupCloneFixture(t)

	remoteBase := filepath.Join(filepath.Dir(coreRepo), "remotes")
	coreRemote := filepath.Join(remoteBase, "core.git")
	userRemote := filepath.Join(remoteBase, "teacher.git")
	for _, remote := range []string{coreRemote, userRemote} {
		if err := os.MkdirAll(remote, 0o755); err != nil {
			t.Fatal(err)
		}
		gitOK(t, remote, "init", "--bare")
	}
	gitOK(t, coreRepo, "remote", "add", "origin", coreRemote)
	p.cfg.WorksheetRemoteBaseURL = remoteBase
	p.cfg.Push = true

	if _, _, err := p.createClones("teacher", "req-7", paths); err != nil {
		t.Fatal(err)
	}
	meta, err := readMetadata(paths)
	if err != nil {
		t.Fatal(err)
	}
	it := store.Request{ID: 7, Kind: store.KindNew, Author: "teacher", Requester: "teacher", Branch: "req-7"}
	return p, it, paths, meta, coreRepo, p.userRepo("teacher")
}

// commitFile adds one committed file to a repository.
func commitFile(t *testing.T, repo, name, content, message string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, name)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOK(t, repo, "add", name)
	gitOK(t, repo, "commit", "-m", message)
}

// agentCommits simulates the agent: one commit in each clone.
func agentCommits(t *testing.T, paths sandboxPaths) {
	t.Helper()
	commitFile(t, paths.Workspace, "agent.txt", "core work", "Agent core work")
	commitFile(t, paths.WorkspaceUserRepo, "sheet.txt", "worksheet work", "Agent worksheet work")
}

// importHeads returns the main tips of the live repositories.
func importHeads(t *testing.T, coreRepo, userRepo string) (string, string) {
	t.Helper()
	return gitOK(t, coreRepo, "rev-parse", "main"), gitOK(t, userRepo, "rev-parse", "main")
}

// assertNoTempArtifacts fails when a live repository still has import temp
// refs, temp branches, or a rebase in progress.
func assertNoTempArtifacts(t *testing.T, repos ...string) {
	t.Helper()
	for _, repo := range repos {
		if out := gitOK(t, repo, "for-each-ref", "refs/learningmaterial/"); out != "" {
			t.Errorf("%s has leftover temp refs: %s", repo, out)
		}
		if out := gitOK(t, repo, "branch", "--list", "learningmaterial/*"); out != "" {
			t.Errorf("%s has leftover temp branches: %s", repo, out)
		}
		gitDir := gitOK(t, repo, "rev-parse", "--git-dir")
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(repo, gitDir)
		}
		for _, dir := range []string{"rebase-merge", "rebase-apply"} {
			if _, err := os.Stat(filepath.Join(gitDir, dir)); !os.IsNotExist(err) {
				t.Errorf("%s has a rebase in progress (%s)", repo, dir)
			}
		}
		if out := gitOK(t, repo, "status", "--porcelain"); out != "" {
			t.Errorf("%s has uncommitted changes: %s", repo, out)
		}
	}
}

func TestImportCommitsHappyPath(t *testing.T) {
	p, it, paths, meta, coreRepo, userRepo := setupImportFixture(t)
	agentCommits(t, paths)
	coreBase, userBase := meta.CoreCommit, meta.WorksheetCommit

	if err := p.importCommits(it, paths, &meta); err != nil {
		t.Fatal(err)
	}

	// Both live mains advanced past their pre-clone tips.
	coreMain, userMain := importHeads(t, coreRepo, userRepo)
	if coreMain == coreBase || userMain == userBase {
		t.Fatalf("live mains did not move: core %q, user %q", coreMain, userMain)
	}
	// The agent's changes landed and the rebased history is linear.
	if got := gitOK(t, coreRepo, "show", "main:agent.txt"); got != "core work" {
		t.Errorf("main:agent.txt = %q", got)
	}
	if got := gitOK(t, userRepo, "show", "main:sheet.txt"); got != "worksheet work" {
		t.Errorf("main:sheet.txt = %q", got)
	}
	if got := gitOK(t, coreRepo, "rev-list", "--count", coreBase+"..main"); got != "1" {
		t.Errorf("core base..main = %s commits, want 1 (linear)", got)
	}
	if got := gitOK(t, userRepo, "rev-list", "--count", userBase+"..main"); got != "1" {
		t.Errorf("user base..main = %s commits, want 1 (linear)", got)
	}
	// The worksheet clone stayed invisible to the core repository.
	if out := gitOK(t, coreRepo, "status", "--porcelain"); out != "" {
		t.Errorf("core repository has uncommitted changes: %s", out)
	}

	// Metadata records the imported tips, and the whole import is now
	// reported as applied.
	if meta.ImportedCoreCommit != coreMain || meta.ImportedWorksheetCommit != userMain {
		t.Errorf("imported tips = (%q, %q), want mains (%q, %q)",
			meta.ImportedCoreCommit, meta.ImportedWorksheetCommit, coreMain, userMain)
	}
	stored, err := readMetadata(paths)
	if err != nil {
		t.Fatal(err)
	}
	if stored.ImportedCoreCommit != coreMain || stored.ImportedWorksheetCommit != userMain {
		t.Errorf("stored imported tips = (%q, %q)", stored.ImportedCoreCommit, stored.ImportedWorksheetCommit)
	}
	applied, err := p.importsApplied(stored)
	if err != nil || !applied {
		t.Errorf("importsApplied = %v, %v; want true, nil", applied, err)
	}

	assertNoTempArtifacts(t, coreRepo, userRepo)

	// The trusted push path publishes both live mains.
	if err := p.push("teacher"); err != nil {
		t.Fatal(err)
	}
	remoteBase := p.cfg.WorksheetRemoteBaseURL
	for _, pair := range [][2]string{
		{coreRepo, filepath.Join(remoteBase, "core.git")},
		{userRepo, filepath.Join(remoteBase, "teacher.git")},
	} {
		local := gitOK(t, pair[0], "rev-parse", "main")
		remote := gitOK(t, pair[1], "rev-parse", "refs/heads/main")
		if local != remote {
			t.Errorf("remote %s is at %s, want %s", pair[1], remote, local)
		}
	}
}

func TestImportCommitsUserCloneOnly(t *testing.T) {
	p, it, paths, meta, coreRepo, userRepo := setupImportFixture(t)
	commitFile(t, paths.WorkspaceUserRepo, "sheet.txt", "worksheet work", "Agent worksheet work")
	coreBase := meta.CoreCommit

	if err := p.importCommits(it, paths, &meta); err != nil {
		t.Fatal(err)
	}
	// The untouched core side is a no-op: the recorded base was already the
	// live main tip, so the imported tip is the base itself.
	if got := gitOK(t, coreRepo, "rev-parse", "main"); got != coreBase {
		t.Errorf("core main = %q, want unchanged base %q", got, coreBase)
	}
	if meta.ImportedCoreCommit != coreBase {
		t.Errorf("ImportedCoreCommit = %q, want base %q", meta.ImportedCoreCommit, coreBase)
	}
	if got := gitOK(t, userRepo, "show", "main:sheet.txt"); got != "worksheet work" {
		t.Errorf("main:sheet.txt = %q", got)
	}
	assertNoTempArtifacts(t, coreRepo, userRepo)
}

func TestImportCommitsNoAgentCommits(t *testing.T) {
	p, it, paths, meta, coreRepo, userRepo := setupImportFixture(t)
	err := p.importCommits(it, paths, &meta)
	if err == nil || !strings.Contains(err.Error(), "no commits") {
		t.Fatalf("importCommits = %v, want an 'agent produced no commits' error", err)
	}
	assertNoTempArtifacts(t, coreRepo, userRepo)
}

func TestImportCommitsConflictLeavesMainUntouched(t *testing.T) {
	p, it, paths, meta, coreRepo, userRepo := setupImportFixture(t)
	// The agent works on content.txt in the core clone...
	commitFile(t, paths.Workspace, "content.txt", "agent version", "Agent core work")
	// ...while live main advances with a conflicting change to the same file.
	commitFile(t, coreRepo, "content.txt", "conflicting live version", "Conflicting live work")
	beforeCore, beforeUser := importHeads(t, coreRepo, userRepo)

	err := p.importCommits(it, paths, &meta)
	if err == nil {
		t.Fatal("importCommits succeeded, want a conflict error")
	}
	if !strings.Contains(err.Error(), "conflict") {
		t.Errorf("error = %v, want it to mention the conflict", err)
	}
	if strings.Contains(err.Error(), "--abort failed") {
		t.Errorf("error = %v, want a clean abort", err)
	}

	// Both live mains are untouched and no temp artifacts remain.
	afterCore, afterUser := importHeads(t, coreRepo, userRepo)
	if afterCore != beforeCore || afterUser != beforeUser {
		t.Errorf("live mains moved: core %q -> %q, user %q -> %q", beforeCore, afterCore, beforeUser, afterUser)
	}
	assertNoTempArtifacts(t, coreRepo, userRepo)
	if meta.ImportedCoreCommit != "" || meta.ImportedWorksheetCommit != "" {
		t.Errorf("metadata recorded imported tips after a failed import: (%q, %q)",
			meta.ImportedCoreCommit, meta.ImportedWorksheetCommit)
	}
}

func TestImportCommitsWorksheetConflictKeepsCoreUntouched(t *testing.T) {
	// The core side is importable and only the worksheet side conflicts: the
	// two-phase commit must leave even the core main untouched.
	p, it, paths, meta, coreRepo, userRepo := setupImportFixture(t)
	agentCommits(t, paths)
	commitFile(t, userRepo, "sheet.txt", "conflicting live version", "Conflicting live work")
	beforeCore, beforeUser := importHeads(t, coreRepo, userRepo)

	err := p.importCommits(it, paths, &meta)
	if err == nil || !strings.Contains(err.Error(), "conflict") {
		t.Fatalf("importCommits = %v, want a conflict error", err)
	}
	afterCore, afterUser := importHeads(t, coreRepo, userRepo)
	if afterCore != beforeCore || afterUser != beforeUser {
		t.Errorf("live mains moved: core %q -> %q, user %q -> %q", beforeCore, afterCore, beforeUser, afterUser)
	}
	assertNoTempArtifacts(t, coreRepo, userRepo)
}

func TestImportCommitsRejectsRewrittenHistory(t *testing.T) {
	p, it, paths, meta, coreRepo, userRepo := setupImportFixture(t)
	// Replace the clone's request branch with an unrelated orphan history.
	gitOK(t, paths.Workspace, "checkout", "--orphan", "rewritten")
	gitOK(t, paths.Workspace, "rm", "-rf", ".")
	commitFile(t, paths.Workspace, "evil.txt", "unrelated", "Unrelated history")
	gitOK(t, paths.Workspace, "branch", "-D", "req-7")
	gitOK(t, paths.Workspace, "branch", "-m", "rewritten", "req-7")
	beforeCore, beforeUser := importHeads(t, coreRepo, userRepo)

	err := p.importCommits(it, paths, &meta)
	if err == nil || !strings.Contains(err.Error(), "descend from the recorded base") {
		t.Fatalf("importCommits = %v, want a non-descendant error", err)
	}
	afterCore, afterUser := importHeads(t, coreRepo, userRepo)
	if afterCore != beforeCore || afterUser != beforeUser {
		t.Errorf("live mains moved: core %q -> %q, user %q -> %q", beforeCore, afterCore, beforeUser, afterUser)
	}
	assertNoTempArtifacts(t, coreRepo, userRepo)
}

func TestImportCommitsRejectsCoreCommitInWorksheetDomain(t *testing.T) {
	p, it, paths, meta, coreRepo, userRepo := setupImportFixture(t)
	// A core commit that touches the nested worksheet repository's domain.
	// The core clone ignores generate/<username>/ (the nested clone lives
	// there) and git add -f silently skips ignored paths, so stage a symlink
	// directly in the index: the imported diff must still name the
	// forbidden path.
	smuggled := filepath.Join(t.TempDir(), "smuggled.txt")
	if err := os.WriteFile(smuggled, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	blob := gitOK(t, paths.Workspace, "hash-object", "-w", smuggled)
	gitOK(t, paths.Workspace, "update-index", "--add", "--cacheinfo", "120000,"+blob+",generate/teacher/smuggled.txt")
	gitOK(t, paths.Workspace, "commit", "-m", "Smuggle worksheet content through the core clone")
	commitFile(t, paths.WorkspaceUserRepo, "sheet.txt", "worksheet work", "Agent worksheet work")
	beforeCore, beforeUser := importHeads(t, coreRepo, userRepo)

	err := p.importCommits(it, paths, &meta)
	if err == nil || !strings.Contains(err.Error(), "generate/teacher") {
		t.Fatalf("importCommits = %v, want a changed-paths error", err)
	}
	afterCore, afterUser := importHeads(t, coreRepo, userRepo)
	if afterCore != beforeCore || afterUser != beforeUser {
		t.Errorf("live mains moved: core %q -> %q, user %q -> %q", beforeCore, afterCore, beforeUser, afterUser)
	}
	assertNoTempArtifacts(t, coreRepo, userRepo)
}

func TestImportCommitsResumesAfterCrash(t *testing.T) {
	p, it, paths, meta, coreRepo, userRepo := setupImportFixture(t)
	agentCommits(t, paths)

	// Simulate a crashed first run: the core side was imported and recorded,
	// the service died before the worksheet side moved.
	if err := p.importCommits(it, paths, &meta); err != nil {
		t.Fatal(err)
	}
	// Rewind only the in-memory metadata to just after the core
	// fast-forward, as if the service had crashed in between.
	meta.ImportedWorksheetCommit = ""
	if err := writeMetadata(paths, meta); err != nil {
		t.Fatal(err)
	}
	coreMain, _ := importHeads(t, coreRepo, userRepo)

	// The retry must redo only the worksheet side. Move the core clone's
	// request branch with a marker commit: if the retry wrongly refetched
	// it, the ancestry validation would fail.
	commitFile(t, paths.Workspace, "marker.txt", "moved meanwhile", "Clone moved after the crash")

	if err := p.importCommits(it, paths, &meta); err != nil {
		t.Fatal(err)
	}
	// No duplicate core commits: still exactly one commit beyond the base.
	if got := gitOK(t, coreRepo, "rev-list", "--count", meta.CoreCommit+"..main"); got != "1" {
		t.Errorf("core base..main = %s commits after the retry, want 1 (no duplicates)", got)
	}
	if got := gitOK(t, coreRepo, "rev-parse", "main"); got != coreMain {
		t.Errorf("core main moved during the retry: %q -> %q", coreMain, got)
	}
	if got := gitOK(t, userRepo, "show", "main:sheet.txt"); got != "worksheet work" {
		t.Errorf("main:sheet.txt = %q after the retry", got)
	}
	applied, err := p.importsApplied(meta)
	if err != nil || !applied {
		t.Errorf("importsApplied = %v, %v; want true, nil", applied, err)
	}
	assertNoTempArtifacts(t, coreRepo, userRepo)
}

func TestImportsAppliedWithoutMetadata(t *testing.T) {
	p, _, _, meta, _, _ := setupImportFixture(t)
	// No import recorded yet: not applied.
	applied, err := p.importsApplied(meta)
	if err != nil || applied {
		t.Errorf("importsApplied = %v, %v; want false, nil", applied, err)
	}
}

// TestMaliciousCloneSymlinks implements the plan's integration test 11: a
// clone whose committed symlinks point at host files must not let any of
// the three phases touch the target:
//
//  1. agent build (inside bubblewrap): writes through the symlink inside
//     the sandbox must fail, because the target is outside every bind;
//  2. import (host-side): fetchAndValidate re-runs validateClone and the
//     sandbox-escape check on the clone path itself, and git never writes
//     through a checked-out symlink (a commit ADDING a symlink changes the
//     link, not the target);
//  3. cleanup (host-side): removeSandbox deletes the link, not the canary.
//
// The canary lives OUTSIDE the sandbox directory in a location shaped like
// /users/<teacher> and stays byte-identical throughout.
func TestMaliciousCloneSymlinks(t *testing.T) {
	p, it, paths, meta, coreRepo, userRepo := setupImportFixture(t)

	// The canary: a temp file OUTSIDE the sandbox dir, next to the live
	// user repository, standing in for a sensitive host file.
	canaryDir := t.TempDir()
	canary := filepath.Join(canaryDir, "passwd-like-canary")
	canaryContent := "must-not-change:0:canary"
	if err := os.WriteFile(canary, []byte(canaryContent), 0o600); err != nil {
		t.Fatal(err)
	}
	assertCanary := func(phase string) {
		t.Helper()
		data, err := os.ReadFile(canary)
		if err != nil {
			t.Fatalf("%s: canary unreadable: %v", phase, err)
		}
		if string(data) != canaryContent {
			t.Fatalf("%s: canary modified through the malicious symlink: %q", phase, data)
		}
	}

	// The agent (running with the sandbox's git identity, as the clones are
	// configured) commits a symlink pointing at the canary in BOTH clones,
	// plus a symlinked generate/<user> parent in the core clone. This is
	// what a malicious or compromised agent could leave behind.
	makeSymlinkCommit := func(repo, linkName, message string) {
		t.Helper()
		link := filepath.Join(repo, linkName)
		if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
			t.Fatal(err)
		}
		os.Remove(link)
		if err := os.Symlink(canary, link); err != nil {
			t.Fatal(err)
		}
		gitOK(t, repo, "add", "-A")
		gitOK(t, repo, "commit", "-m", message)
	}
	makeSymlinkCommit(paths.Workspace, "escape.txt", "Add a symlink to a host file")
	makeSymlinkCommit(paths.WorkspaceUserRepo, "escape.txt", "Add a symlink to a host file")

	// Phase 1: agent build INSIDE the real bubblewrap policy. Reading or
	// writing through the committed symlink must fail (the target is not
	// mounted), and the canary must not change.
	if _, err := exec.LookPath("bwrap"); err == nil && !testing.Short() {
		spec := sandbox.Spec{
			Name:      "req-7-symlink-probe",
			Workspace: paths.Workspace,
			Home:      paths.Home,
			State:     paths.State,
			Tmp:       paths.Tmp,
			GoCache:   paths.GoCache,
			ModCache:  "-",
			Command: []string{"/bin/bash", "-c", strings.Join([]string{
				"if cat escape.txt >/dev/null 2>&1; then echo read-escaped; else echo read-blocked; fi",
				"if echo pwned >> escape.txt 2>/dev/null; then echo write-escaped; else echo write-blocked; fi",
				"if cat generate/teacher/escape.txt >/dev/null 2>&1; then echo nested-read-escaped; else echo nested-read-blocked; fi",
				"echo pwned >> generate/teacher/escape.txt 2>/dev/null && echo nested-write-escaped || echo nested-write-blocked",
			}, " ; ")},
		}
		h, err := sandbox.Start(context.Background(), spec)
		if err != nil {
			t.Fatalf("start the symlink probe sandbox: %v", err)
		}
		_ = h.Wait()
		out := h.OutputString()
		t.Logf("symlink probe output:\n%s", out)
		for _, want := range []string{"read-blocked", "write-blocked", "nested-read-blocked", "nested-write-blocked"} {
			if !strings.Contains(out, want) {
				t.Errorf("symlink probe: missing %q (the sandbox reached the canary)", want)
			}
		}
		assertCanary("agent build (in-sandbox)")
	} else {
		t.Log("bwrap unavailable or -short: skipping the in-sandbox phase")
	}

	// Phase 2: import (host-side). The agent's symlink commits are
	// legitimate git objects, so the import itself accepts the tips (they
	// descend from the recorded bases); importing must not create, follow,
	// or delete through the links in the LIVE repositories: git updates the
	// link entry, never the target. Then the live checkouts, when the
	// branch is materialized (rebase + checkout), contain a dangling link
	// whose target is unchanged.
	if err := p.importCommits(it, paths, &meta); err != nil {
		t.Fatalf("import of symlink commits: %v", err)
	}
	assertCanary("import")
	for _, repo := range []string{coreRepo, userRepo} {
		// The live repository now contains the symlink as a git entry ...
		if got := gitOK(t, repo, "cat-file", "-t", "main:escape.txt"); got != "blob" {
			t.Errorf("%s main:escape.txt type = %q, want blob (a symlink)", repo, got)
		}
		// ... and the checkout on disk is a symlink whose TARGET TEXT is the
		// canary path, but the canary itself is untouched because nothing
		// opened the link for writing. git status must be clean: a write
		// through the link would show up as a modification of the target or
		// a dirty tree.
		if out := gitOK(t, repo, "status", "--porcelain"); out != "" {
			t.Errorf("%s is dirty after the import: %s", repo, out)
		}
	}
	assertCanary("import (live checkouts materialized)")

	// A clone PATH that escapes through a symlinked parent is rejected
	// outright before any fetch (validateClone), and one outside the
	// sandbox root is rejected by fetchAndValidate's pathBelow check. Build
	// a sibling directory that pretends to be the workspace through a link.
	elsewhere := t.TempDir()
	fakeClone := filepath.Join(elsewhere, "workspace")
	if err := p.cloneStandalone(coreRepo, fakeClone, "req-7", "sandbox-req-7@localhost"); err != nil {
		t.Fatal(err)
	}
	linkRoot := filepath.Join(elsewhere, "req-7-link")
	if err := os.Symlink(paths.Root, linkRoot); err != nil {
		t.Fatal(err)
	}
	linkedPaths := paths
	linkedPaths.Root = linkRoot
	linkedPaths.Workspace = filepath.Join(linkRoot, "workspace")
	linkedPaths.WorkspaceUserRepo = filepath.Join(linkRoot, "workspace", "generate", "teacher")
	if err := validateClone(linkedPaths.Workspace); err == nil {
		t.Error("validateClone accepted a workspace reached through a symlinked sandbox root")
	}
	r := &importRepo{name: "core", live: coreRepo, clone: linkedPaths.Workspace, base: meta.CoreCommit}
	if err := p.fetchAndValidate(it, r, &meta, linkedPaths, "teacher", nil); err == nil {
		t.Error("fetchAndValidate accepted a clone path escaping the sandbox root")
	}
	assertCanary("import path validation")

	// Phase 3: cleanup (host-side). removeSandbox removes the whole
	// req-<id> directory; the symlinks inside it (including generate/teacher
	// itself, if an agent replaced it with a link) must be deleted AS LINKS.
	// Replace the nested clone path's parent with a symlink to the canary
	// dir to simulate the worst case, then remove.
	//
	// (os.RemoveAll never follows symlinks; this test pins that behavior.)
	escapeRoot := t.TempDir()
	escapePaths, err := sandboxPathsFor(escapeRoot, 21, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(escapePaths.Workspace), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(canaryDir, escapePaths.Workspace); err != nil {
		t.Fatal(err)
	}
	pp := New(nil, Config{Sandbox: true, SandboxRoot: escapeRoot})
	pp.removeSandbox(21)
	if _, err := os.Lstat(escapePaths.Root); !os.IsNotExist(err) {
		t.Errorf("sandbox root still exists after removeSandbox")
	}
	assertCanary("cleanup")
	if entries, err := os.ReadDir(canaryDir); err != nil || len(entries) != 1 {
		t.Errorf("cleanup removed canary dir contents: entries=%d err=%v", len(entries), err)
	}
}
