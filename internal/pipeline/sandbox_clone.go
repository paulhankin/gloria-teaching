package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"learningmaterial/internal/store"
)

// sandboxFormatVersion is the current sandbox metadata format. Sandboxes with
// a different version are not reused after a restart.
const sandboxFormatVersion = 1

// Sandbox lifecycle phases recorded in metadata.json.
const (
	phaseCreating      = "creating"
	phaseReady         = "ready"
	phaseAgentRunning  = "agent-running"
	phaseAgentFinished = "agent-finished"
	phaseValidating    = "validating"
	phaseImporting     = "importing"
	phasePublished     = "published"
	phaseCleaning      = "cleaning"
	phaseCleaned       = "cleaned"
)

// sandboxMetadata is written to state/metadata.json. It allows the trusted
// pipeline to recover a sandbox after a service restart without new database
// columns.
type sandboxMetadata struct {
	RequestID       int64     `json:"request_id"`
	Username        string    `json:"username"`
	Branch          string    `json:"branch"`
	Version         int       `json:"version"`
	CreatedAt       time.Time `json:"created_at"`
	CoreCommit      string    `json:"core_commit"`
	WorksheetCommit string    `json:"worksheet_commit"`
	Phase           string    `json:"phase"`
	ConversationID  string    `json:"conversation_id"`
	PID             int       `json:"pid"`  // 0 while no isolated server is running
	Unit            string    `json:"unit"` // transient systemd unit/scope name while active
	CleanupState    string    `json:"cleanup_state"`

	// ImportedCoreCommit and ImportedWorksheetCommit record the rebased tips
	// that were fast-forwarded into each live main. They make the import
	// idempotent: after a crash between the two fast-forwards (or before the
	// push) the already-imported side is skipped instead of duplicated.
	ImportedCoreCommit      string `json:"imported_core_commit,omitempty"`
	ImportedWorksheetCommit string `json:"imported_worksheet_commit,omitempty"`
}

// writeMetadata atomically replaces the metadata file (temp file + rename).
func writeMetadata(paths sandboxPaths, meta sandboxMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("encode sandbox metadata: %w", err)
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(paths.Metadata), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(paths.Metadata), ".metadata-*.json")
	if err != nil {
		return fmt.Errorf("create sandbox metadata temp file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write sandbox metadata: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync sandbox metadata: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close sandbox metadata: %w", err)
	}
	if err := os.Rename(tmpName, paths.Metadata); err != nil {
		return fmt.Errorf("replace sandbox metadata: %w", err)
	}
	return nil
}

// readMetadata loads the sandbox metadata file.
func readMetadata(paths sandboxPaths) (sandboxMetadata, error) {
	var meta sandboxMetadata
	data, err := os.ReadFile(paths.Metadata)
	if err != nil {
		return meta, fmt.Errorf("read sandbox metadata: %w", err)
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return meta, fmt.Errorf("parse sandbox metadata: %w", err)
	}
	return meta, nil
}

// setPhase updates the lifecycle phase in the stored metadata. Leaving the
// agent-running phase clears the recorded process bookkeeping.
func setPhase(paths sandboxPaths, phase string) error {
	meta, err := readMetadata(paths)
	if err != nil {
		return err
	}
	meta.Phase = phase
	if phase != phaseAgentRunning {
		meta.PID = 0
		meta.Unit = ""
	}
	return writeMetadata(paths, meta)
}

// --- clones ----------------------------------------------------------------

// createClones replaces the sandbox workspace with standalone clones of the
// live core repository and the requester's worksheet repository, checks out
// the request branch in both, removes all remotes, and records the base
// commits in the sandbox metadata.
func (p *Pipeline) createClones(username, branch string, paths sandboxPaths) (baseCore, baseUser string, err error) {
	if err := p.ensureUserRepo(username); err != nil {
		return "", "", fmt.Errorf("worksheet repository: %w", err)
	}
	if err := os.MkdirAll(paths.State, 0o755); err != nil {
		return "", "", err
	}
	meta := sandboxMetadata{
		RequestID: sandboxRequestID(paths),
		Username:  username,
		Branch:    branch,
		Version:   sandboxFormatVersion,
		CreatedAt: time.Now().UTC(),
		Phase:     phaseCreating,
	}
	if err := writeMetadata(paths, meta); err != nil {
		return "", "", err
	}

	os.RemoveAll(paths.Workspace)
	email := "sandbox-" + branch + "@localhost"
	if err := p.cloneStandalone(p.cfg.Repo, paths.Workspace, branch, email); err != nil {
		return "", "", fmt.Errorf("clone core repository: %w", err)
	}
	if err := p.cloneStandalone(p.userRepo(username), paths.WorkspaceUserRepo, branch, email); err != nil {
		return "", "", fmt.Errorf("clone worksheet repository: %w", err)
	}
	if err := validateClone(paths.Workspace); err != nil {
		return "", "", fmt.Errorf("core clone: %w", err)
	}
	if err := validateClone(paths.WorkspaceUserRepo); err != nil {
		return "", "", fmt.Errorf("worksheet clone: %w", err)
	}

	if baseCore, err = p.headCommit(paths.Workspace); err != nil {
		return "", "", fmt.Errorf("record core base commit: %w", err)
	}
	if baseUser, err = p.headCommit(paths.WorkspaceUserRepo); err != nil {
		return "", "", fmt.Errorf("record worksheet base commit: %w", err)
	}
	meta.CoreCommit = baseCore
	meta.WorksheetCommit = baseUser
	meta.Phase = phaseReady
	if err := writeMetadata(paths, meta); err != nil {
		return "", "", err
	}
	return baseCore, baseUser, nil
}

// sandboxRequestID recovers the request ID from a validated sandbox path.
func sandboxRequestID(paths sandboxPaths) int64 {
	var id int64
	if n, err := fmt.Sscanf(filepath.Base(paths.Root), "req-%d", &id); err == nil && n == 1 {
		return id
	}
	return 0
}

// cloneStandalone clones src into dst as a fully independent repository:
// --no-local forces a real copy instead of hardlinks, and no --shared is
// used, so no Git metadata of the live repository is referenced.
func (p *Pipeline) cloneStandalone(src, dst, branch, email string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if _, err := p.git(filepath.Dir(dst), "clone", "--no-hardlinks", "--no-local", src, dst); err != nil {
		return err
	}
	if _, err := p.git(dst, "checkout", "-b", branch); err != nil {
		return err
	}
	if _, err := p.git(dst, "config", "user.name", "Learning Material Sandbox"); err != nil {
		return err
	}
	if _, err := p.git(dst, "config", "user.email", email); err != nil {
		return err
	}
	out, err := p.git(dst, "remote")
	if err != nil {
		return err
	}
	for _, remote := range strings.Fields(out) {
		if _, err := p.git(dst, "remote", "remove", remote); err != nil {
			return err
		}
	}
	return nil
}

// headCommit returns the current HEAD commit of the repository at dir.
func (p *Pipeline) headCommit(dir string) (string, error) {
	out, err := p.git(dir, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// validateClone re-checks that dir is a self-contained Git repository with no
// remotes whose Git metadata lives inside the clone root. It rejects linked
// worktrees, --shared clones, and clone paths that move when symlinks are
// resolved (a symlinked parent component could smuggle the agent's working
// tree outside the sandbox). Used at creation time and again before import.
func validateClone(dir string) error {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	root, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return fmt.Errorf("%s: %w", dir, err)
	}
	if root != filepath.Clean(abs) {
		return fmt.Errorf("clone path %s escapes through a symlink to %s", dir, root)
	}
	info, err := os.Stat(filepath.Join(dir, ".git"))
	if err != nil {
		return fmt.Errorf("not a git repository: %s", dir)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is a linked worktree, not a standalone clone", dir)
	}
	for _, what := range [][2]string{
		{"--git-dir", "git dir"},
		{"--git-common-dir", "git common dir"},
	} {
		out, err := gitOutput(dir, "rev-parse", what[0])
		if err != nil {
			return err
		}
		gitDir := strings.TrimSpace(out)
		if !filepath.IsAbs(gitDir) {
			gitDir = filepath.Join(dir, gitDir)
		}
		resolvedGitDir, err := filepath.EvalSymlinks(gitDir)
		if err != nil {
			return fmt.Errorf("%s: %s: %w", dir, what[1], err)
		}
		if !pathBelow(root, filepath.Clean(resolvedGitDir)) {
			return fmt.Errorf("%s: %s %s escapes the clone root %s", dir, what[1], resolvedGitDir, root)
		}
	}
	out, err := gitOutput(dir, "remote")
	if err != nil {
		return err
	}
	if remotes := strings.Fields(out); len(remotes) > 0 {
		return fmt.Errorf("%s still has remotes configured: %s", dir, strings.Join(remotes, ", "))
	}
	return nil
}

// gitOutput runs a read-only git command in dir without a Pipeline receiver
// so validateClone is usable in tests and future import code alike.
func gitOutput(dir string, args ...string) (string, error) {
	return (&Pipeline{}).git(dir, args...)
}

// cloneAhead reports whether either sandbox clone has commits beyond its
// recorded base commit. It replaces the worktree-mode main..branch comparison.
func (p *Pipeline) cloneAhead(paths sandboxPaths, meta sandboxMetadata) (bool, error) {
	checks := []struct {
		dir  string
		base string
	}{
		{paths.Workspace, meta.CoreCommit},
		{paths.WorkspaceUserRepo, meta.WorksheetCommit},
	}
	for _, check := range checks {
		if check.base == "" {
			return false, fmt.Errorf("no base commit recorded for %s", check.dir)
		}
		head, err := p.headCommit(check.dir)
		if err != nil {
			return false, err
		}
		if head != check.base {
			return true, nil
		}
	}
	return false, nil
}

// --- cleanup ---------------------------------------------------------------

// removeSandbox deletes the whole request sandbox directory. Process
// termination is added by a later step; for now only the directory is
// removed. Preview dirs under PreviewRoot are handled separately.
func (p *Pipeline) removeSandbox(id int64) {
	root := p.sandboxRoot(id)
	if root == "" {
		return
	}
	if err := os.RemoveAll(root); err != nil {
		p.Logf("#%d: remove sandbox %s: %v", id, root, err)
	}
}

// sandboxRoot returns the sandbox directory of one request, or "" when the
// sandbox root is not configured or not absolute.
func (p *Pipeline) sandboxRoot(id int64) string {
	if p.cfg.SandboxRoot == "" || !filepath.IsAbs(p.cfg.SandboxRoot) {
		return ""
	}
	return filepath.Join(p.cfg.SandboxRoot, fmt.Sprintf("req-%d", id))
}

// cleanupWorkspace removes the agent workspace of a finished or rejected
// request: the whole sandbox in sandbox mode, the linked worktrees otherwise.
// The request's preview directory is removed in both modes.
func (p *Pipeline) cleanupWorkspace(it store.Request) {
	if p.cfg.Sandbox {
		p.removeSandbox(it.ID)
		os.RemoveAll(p.previewDir(it.ID))
		p.db.SetPreview(it.ID, false)
		return
	}
	p.removeWorktree(it)
}
