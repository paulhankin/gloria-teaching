package pipeline

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func gitOK(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git -C %s %v: %v: %s", dir, args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// initTestRepo creates a git repository with one commit and returns its HEAD.
func initTestRepo(t *testing.T, dir, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitOK(t, dir, "init", "-b", "main")
	gitOK(t, dir, "config", "user.name", "Test")
	gitOK(t, dir, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(dir, "content.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOK(t, dir, "add", ".")
	gitOK(t, dir, "commit", "-m", "Initial")
	return gitOK(t, dir, "rev-parse", "HEAD")
}

func TestSandboxPathsFor(t *testing.T) {
	root := filepath.Join(t.TempDir(), "sandboxes")
	paths, err := sandboxPathsFor(root, 42, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	wantRoot := filepath.Join(root, "req-42")
	if paths.Root != wantRoot {
		t.Fatalf("Root = %q, want %q", paths.Root, wantRoot)
	}
	// Every derived path must stay below the sandbox root.
	all := map[string]string{
		"Workspace":         paths.Workspace,
		"WorkspaceUserRepo": paths.WorkspaceUserRepo,
		"Home":              paths.Home,
		"GoCache":           paths.GoCache,
		"GoConfig":          paths.GoConfig,
		"State":             paths.State,
		"DB":                paths.DB,
		"Socket":            paths.Socket,
		"ServerLog":         paths.ServerLog,
		"Metadata":          paths.Metadata,
		"Tmp":               paths.Tmp,
	}
	for name, path := range all {
		if !pathBelow(paths.Root, path) {
			t.Errorf("%s = %q is not below sandbox root %q", name, path, paths.Root)
		}
		if !filepath.IsAbs(path) {
			t.Errorf("%s = %q is not absolute", name, path)
		}
	}
	// Spot-check the layout from the plan.
	if paths.Workspace != filepath.Join(wantRoot, "workspace") {
		t.Errorf("Workspace = %q", paths.Workspace)
	}
	if paths.WorkspaceUserRepo != filepath.Join(wantRoot, "workspace", "generate", "teacher") {
		t.Errorf("WorkspaceUserRepo = %q", paths.WorkspaceUserRepo)
	}
	if paths.DB != filepath.Join(wantRoot, "state", "shelley.db") {
		t.Errorf("DB = %q", paths.DB)
	}
	if paths.Socket != filepath.Join(wantRoot, "state", "shelley.sock") {
		t.Errorf("Socket = %q", paths.Socket)
	}
	if paths.ServerLog != filepath.Join(wantRoot, "state", "server.log") {
		t.Errorf("ServerLog = %q", paths.ServerLog)
	}
	if paths.Metadata != filepath.Join(wantRoot, "state", "metadata.json") {
		t.Errorf("Metadata = %q", paths.Metadata)
	}
	if paths.GoCache != filepath.Join(wantRoot, "home", ".cache", "go-build") {
		t.Errorf("GoCache = %q", paths.GoCache)
	}
	if paths.Tmp != filepath.Join(wantRoot, "tmp") {
		t.Errorf("Tmp = %q", paths.Tmp)
	}
}

func TestSandboxPathsForRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	for _, username := range []string{"", ".", "..", "a/b", `a\b`, "../x", "./x", "a//b"} {
		if _, err := sandboxPathsFor(root, 1, username); err == nil {
			t.Errorf("sandboxPathsFor(_, 1, %q) succeeded, want error", username)
		}
	}
	// A relative sandbox root is rejected.
	if _, err := sandboxPathsFor("data/sandboxes", 1, "teacher"); err == nil {
		t.Error("sandboxPathsFor with relative root succeeded, want error")
	}
	// An empty root is rejected.
	if _, err := sandboxPathsFor("", 1, "teacher"); err == nil {
		t.Error("sandboxPathsFor with empty root succeeded, want error")
	}
}

// setupCloneFixture creates a live core repo and a live user repo and returns
// a pipeline and sandbox paths ready for createClones.
func setupCloneFixture(t *testing.T) (*Pipeline, sandboxPaths, string, string, string) {
	t.Helper()
	root := t.TempDir()
	coreRepo := filepath.Join(root, "live-core")
	coreHead := initTestRepo(t, coreRepo, "framework")
	// The core repository ignores the whole generate/ tree: per-user
	// worksheet repositories are mounted below it.
	if err := os.WriteFile(filepath.Join(coreRepo, ".gitignore"), []byte("/generate/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOK(t, coreRepo, "add", ".")
	gitOK(t, coreRepo, "commit", "-m", "Ignore generate")
	coreHead = gitOK(t, coreRepo, "rev-parse", "HEAD")

	usersRoot := filepath.Join(root, "users")
	userRepo := filepath.Join(usersRoot, "teacher")
	userHead := initTestRepo(t, userRepo, "worksheet")

	p := &Pipeline{cfg: Config{
		Repo:          coreRepo,
		WorksheetRoot: usersRoot,
		Sandbox:       true,
		SandboxRoot:   filepath.Join(root, "sandboxes"),
	}}
	paths, err := sandboxPathsFor(p.cfg.SandboxRoot, 7, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	return p, paths, coreRepo, coreHead, userHead
}

func TestCreateClones(t *testing.T) {
	p, paths, coreRepo, coreHead, userHead := setupCloneFixture(t)

	baseCore, baseUser, err := p.createClones("teacher", "req-7", paths)
	if err != nil {
		t.Fatal(err)
	}
	if baseCore != coreHead {
		t.Errorf("base core commit = %q, want live HEAD %q", baseCore, coreHead)
	}
	if baseUser != userHead {
		t.Errorf("base worksheet commit = %q, want live HEAD %q", baseUser, userHead)
	}

	for _, clone := range []string{paths.Workspace, paths.WorkspaceUserRepo} {
		if got := gitOK(t, clone, "branch", "--show-current"); got != "req-7" {
			t.Errorf("%s branch = %q, want req-7", clone, got)
		}
		if got := gitOK(t, clone, "remote"); got != "" {
			t.Errorf("%s remotes = %q, want none", clone, got)
		}
		if got := gitOK(t, clone, "config", "user.name"); got != "Learning Material Sandbox" {
			t.Errorf("%s user.name = %q", clone, got)
		}
		if got := gitOK(t, clone, "config", "user.email"); got != "sandbox-req-7@localhost" {
			t.Errorf("%s user.email = %q", clone, got)
		}
		// The Git metadata must live inside the clone, not in the live repo.
		common := gitOK(t, clone, "rev-parse", "--git-common-dir")
		if !filepath.IsAbs(common) {
			common = filepath.Join(clone, common)
		}
		resolved, err := filepath.EvalSymlinks(common)
		if err != nil {
			t.Fatal(err)
		}
		if !pathBelow(clone, resolved) {
			t.Errorf("%s git common dir resolves to %q outside the clone", clone, resolved)
		}
		if strings.HasPrefix(resolved, coreRepo) {
			t.Errorf("%s git common dir resolves into the live repository", clone)
		}
	}

	// Metadata records the base commits and the ready phase.
	meta, err := readMetadata(paths)
	if err != nil {
		t.Fatal(err)
	}
	if meta.RequestID != 7 || meta.Username != "teacher" || meta.Branch != "req-7" {
		t.Errorf("metadata identity = %#v", meta)
	}
	if meta.Version != sandboxFormatVersion {
		t.Errorf("metadata version = %d, want %d", meta.Version, sandboxFormatVersion)
	}
	if meta.CoreCommit != coreHead || meta.WorksheetCommit != userHead {
		t.Errorf("metadata commits = (%q, %q)", meta.CoreCommit, meta.WorksheetCommit)
	}
	if meta.Phase != phaseReady {
		t.Errorf("metadata phase = %q, want %q", meta.Phase, phaseReady)
	}

	// The clones are independent of the live repositories.
	if err := os.WriteFile(filepath.Join(coreRepo, "live-only.txt"), []byte("live"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOK(t, coreRepo, "add", ".")
	gitOK(t, coreRepo, "commit", "-m", "Live work after cloning")
	if _, err := os.Stat(filepath.Join(paths.Workspace, "live-only.txt")); !os.IsNotExist(err) {
		t.Errorf("live commit appeared in the clone")
	}
	if got := gitOK(t, paths.Workspace, "rev-parse", "HEAD"); got != coreHead {
		t.Errorf("clone HEAD = %q moved with the live repository", got)
	}

	// Writes inside the clone must not touch the live repository.
	if err := os.WriteFile(filepath.Join(paths.Workspace, "clone-only.txt"), []byte("clone"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitOK(t, paths.Workspace, "add", ".")
	gitOK(t, paths.Workspace, "commit", "-m", "Agent work")
	if _, err := os.Stat(filepath.Join(coreRepo, "clone-only.txt")); !os.IsNotExist(err) {
		t.Errorf("clone file appeared in the live repository")
	}

	// cloneAhead: the core clone moved ahead, the worksheet clone did not.
	ahead, err := p.cloneAhead(paths, meta)
	if err != nil {
		t.Fatal(err)
	}
	if !ahead {
		t.Error("cloneAhead = false after a clone commit")
	}
	if _, err := p.headCommit(paths.WorkspaceUserRepo); err != nil {
		t.Fatal(err)
	}
	if got := gitOK(t, paths.WorkspaceUserRepo, "rev-parse", "HEAD"); got != userHead {
		t.Errorf("worksheet clone HEAD = %q, want unchanged %q", got, userHead)
	}
}

func TestCloneAheadNoCommits(t *testing.T) {
	p, paths, _, _, _ := setupCloneFixture(t)
	if _, _, err := p.createClones("teacher", "req-7", paths); err != nil {
		t.Fatal(err)
	}
	meta, err := readMetadata(paths)
	if err != nil {
		t.Fatal(err)
	}
	ahead, err := p.cloneAhead(paths, meta)
	if err != nil {
		t.Fatal(err)
	}
	if ahead {
		t.Error("cloneAhead = true for untouched clones")
	}
}

func TestValidateClone(t *testing.T) {
	p, paths, coreRepo, _, _ := setupCloneFixture(t)
	if _, _, err := p.createClones("teacher", "req-7", paths); err != nil {
		t.Fatal(err)
	}

	// A fresh standalone clone passes.
	if err := validateClone(paths.Workspace); err != nil {
		t.Fatalf("validateClone(standalone clone): %v", err)
	}
	if err := validateClone(paths.WorkspaceUserRepo); err != nil {
		t.Fatalf("validateClone(nested clone): %v", err)
	}

	root := t.TempDir()

	// A linked worktree (its .git is a file pointing at the live repository)
	// must be rejected.
	worktreeDir := filepath.Join(root, "linked-worktree")
	gitOK(t, coreRepo, "worktree", "add", "-b", "wt-test", worktreeDir, "main")
	if err := validateClone(worktreeDir); err == nil {
		t.Error("validateClone accepted a linked worktree")
	}

	// A repository with a remote must be rejected.
	remoteClone := filepath.Join(root, "remote-clone")
	if err := p.cloneStandalone(coreRepo, remoteClone, "req-8", "sandbox-req-8@localhost"); err != nil {
		t.Fatal(err)
	}
	gitOK(t, remoteClone, "remote", "add", "origin", coreRepo)
	if err := validateClone(remoteClone); err == nil {
		t.Error("validateClone accepted a clone with a remote")
	}

	// A symlink escape: a parent component of the clone path is a symlink to
	// somewhere else.
	elsewhere := filepath.Join(root, "elsewhere")
	if err := os.MkdirAll(elsewhere, 0o755); err != nil {
		t.Fatal(err)
	}
	escapedClone := filepath.Join(elsewhere, "escaped")
	if err := p.cloneStandalone(coreRepo, escapedClone, "req-9", "sandbox-req-9@localhost"); err != nil {
		t.Fatal(err)
	}
	linkParent := filepath.Join(root, "link-parent")
	if err := os.Symlink(elsewhere, linkParent); err != nil {
		t.Fatal(err)
	}
	if err := validateClone(filepath.Join(linkParent, "escaped")); err == nil {
		t.Error("validateClone accepted a symlink escape")
	}

	// A plain directory must be rejected.
	plain := filepath.Join(root, "plain")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateClone(plain); err == nil {
		t.Error("validateClone accepted a non-repository")
	}
}

func TestSandboxMetadataRoundTrip(t *testing.T) {
	root := t.TempDir()
	paths, err := sandboxPathsFor(root, 9, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	meta := sandboxMetadata{
		RequestID:       9,
		Username:        "teacher",
		Branch:          "req-9",
		Version:         sandboxFormatVersion,
		CoreCommit:      "0123456789abcdef0123456789abcdef01234567",
		WorksheetCommit: "abcdef0123456789abcdef0123456789abcdef01",
		Phase:           phaseReady,
		ConversationID:  "conv-1",
		PID:             1234,
		Unit:            "lm-sandbox-9.scope",
		CleanupState:    "pending",
	}
	if err := writeMetadata(paths, meta); err != nil {
		t.Fatal(err)
	}
	got, err := readMetadata(paths)
	if err != nil {
		t.Fatal(err)
	}
	if got != meta {
		t.Fatalf("readMetadata = %#v, want %#v", got, meta)
	}

	// setPhase updates the phase and clears the process bookkeeping once the
	// agent is no longer running.
	if err := setPhase(paths, phaseAgentFinished); err != nil {
		t.Fatal(err)
	}
	got, err = readMetadata(paths)
	if err != nil {
		t.Fatal(err)
	}
	if got.Phase != phaseAgentFinished {
		t.Fatalf("phase = %q, want %q", got.Phase, phaseAgentFinished)
	}
	if got.PID != 0 || got.Unit != "" {
		t.Fatalf("process bookkeeping not cleared: pid=%d unit=%q", got.PID, got.Unit)
	}
}

func TestWriteMetadataAtomic(t *testing.T) {
	root := t.TempDir()
	paths, err := sandboxPathsFor(root, 3, "teacher")
	if err != nil {
		t.Fatal(err)
	}

	// Concurrent writes must never leave a torn or missing metadata file.
	done := make(chan error, 2)
	for worker := 0; worker < 2; worker++ {
		go func(worker int) {
			var err error
			for i := 0; i < 20; i++ {
				err = writeMetadata(paths, sandboxMetadata{
					RequestID: 3,
					Username:  "teacher",
					Branch:    "req-3",
					Version:   sandboxFormatVersion,
					Phase:     fmt.Sprintf("phase-%d-%d", worker, i),
				})
				if err != nil {
					break
				}
			}
			done <- err
		}(worker)
	}
	for worker := 0; worker < 2; worker++ {
		if err := <-done; err != nil {
			t.Fatal(err)
		}
	}

	// Every read during and after the writes sees complete valid JSON.
	data, err := os.ReadFile(paths.Metadata)
	if err != nil {
		t.Fatal(err)
	}
	var meta sandboxMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatalf("metadata is not valid JSON: %v", err)
	}
	if !strings.HasPrefix(meta.Phase, "phase-") {
		t.Fatalf("metadata phase = %q", meta.Phase)
	}

	// No temp files are left behind.
	entries, err := os.ReadDir(paths.State)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.Name() != "metadata.json" {
			t.Errorf("unexpected file in state dir: %s", entry.Name())
		}
	}
}

func TestRemoveSandbox(t *testing.T) {
	root := t.TempDir()
	p := &Pipeline{cfg: Config{Sandbox: true, SandboxRoot: root}}
	paths, err := sandboxPathsFor(root, 5, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.State, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paths.Metadata, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	p.removeSandbox(5)
	if _, err := os.Stat(paths.Root); !os.IsNotExist(err) {
		t.Errorf("sandbox root still exists after removeSandbox")
	}
	// Removing a missing sandbox is fine.
	p.removeSandbox(5)

	// A missing or relative sandbox root is a no-op, never a deletion.
	p.cfg.SandboxRoot = ""
	p.removeSandbox(5)
	p.cfg.SandboxRoot = "data/sandboxes"
	p.removeSandbox(5)
}

func TestSandboxLimitsDefaults(t *testing.T) {
	limits := SandboxLimits{}.withDefaults()
	if limits.MemoryMax != "4G" {
		t.Errorf("MemoryMax = %q", limits.MemoryMax)
	}
	if limits.TasksMax != 512 {
		t.Errorf("TasksMax = %d", limits.TasksMax)
	}
	if limits.CPUQuota != "200%" {
		t.Errorf("CPUQuota = %q", limits.CPUQuota)
	}
	if limits.RuntimeMax != 90*time.Minute {
		t.Errorf("RuntimeMax = %v", limits.RuntimeMax)
	}
	if limits.GracefulStop != 20*time.Second {
		t.Errorf("GracefulStop = %v", limits.GracefulStop)
	}
	if limits.WorkspaceMaxBytes != 2<<30 {
		t.Errorf("WorkspaceMaxBytes = %d", limits.WorkspaceMaxBytes)
	}
	if limits.MaxSandboxes != 2 {
		t.Errorf("MaxSandboxes = %d", limits.MaxSandboxes)
	}

	// Explicit values are kept.
	custom := SandboxLimits{MemoryMax: "8G", TasksMax: 64, CPUQuota: "100%",
		RuntimeMax: 5 * time.Minute, GracefulStop: time.Second, WorkspaceMaxBytes: 1 << 20, MaxSandboxes: 4}
	if got := custom.withDefaults(); got != custom {
		t.Errorf("withDefaults changed explicit limits: %#v", got)
	}

	// New normalizes the config.
	p := New(nil, Config{})
	if p.cfg.Limits.MaxSandboxes != 2 {
		t.Errorf("New did not normalize limits: %#v", p.cfg.Limits)
	}
}
