package pipeline

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"learningmaterial/internal/sandbox"
	"learningmaterial/internal/store"
)

func TestDirSize(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.txt"), make([]byte, 100), 0o644); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(root, "sub")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub, "b.txt"), make([]byte, 200), 0o644); err != nil {
		t.Fatal(err)
	}
	size, err := dirSize(root)
	if err != nil {
		t.Fatal(err)
	}
	if size < 300 {
		t.Fatalf("dirSize = %d, want at least the 300 file bytes", size)
	}

	// A symlink to a large file outside the root contributes its own entry
	// size, not the target's: WalkDir never follows symlinks.
	outside := t.TempDir()
	big := filepath.Join(outside, "big.bin")
	if err := os.WriteFile(big, make([]byte, 10_000), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(big, filepath.Join(sub, "link")); err != nil {
		t.Fatal(err)
	}
	withLink, err := dirSize(root)
	if err != nil {
		t.Fatal(err)
	}
	if withLink >= size+10_000 {
		t.Fatalf("dirSize followed the symlink: %d -> %d", size, withLink)
	}
	if withLink <= size {
		t.Fatalf("dirSize ignored the symlink entry itself: %d -> %d", size, withLink)
	}

	// A missing root counts as zero bytes and reports the stat error.
	size2, err := dirSize(filepath.Join(root, "gone"))
	t.Logf("dirSize(missing) = %d, %v", size2, err)
	if size2 != 0 || err == nil || !os.IsNotExist(err) {
		t.Fatalf("dirSize(missing) = %d, %v; want 0 and an os.IsNotExist error", size2, err)
	}
}

func TestEnforceWorkspaceLimit(t *testing.T) {
	root := t.TempDir()
	paths, err := sandboxPathsFor(root, 11, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.State, 0o755); err != nil {
		t.Fatal(err)
	}
	payload := filepath.Join(paths.State, "blob.bin")
	if err := os.WriteFile(payload, make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}

	p := New(nil, Config{Sandbox: true, SandboxRoot: root})

	// Under the limit: fine.
	if err := p.enforceWorkspaceLimit(paths, 1<<20); err != nil {
		t.Fatalf("enforceWorkspaceLimit under the limit: %v", err)
	}
	// Over the limit: the failure names the limit and the request.
	err = p.enforceWorkspaceLimit(paths, 1024)
	if err == nil {
		t.Fatal("enforceWorkspaceLimit over the limit returned nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "1024 byte limit") || !strings.Contains(msg, "#11") {
		t.Fatalf("failure does not mention the limit and the request: %q", msg)
	}
	// Disabled limit and the worktree-mode zero paths pass.
	if err := p.enforceWorkspaceLimit(paths, 0); err != nil {
		t.Fatalf("enforceWorkspaceLimit with a disabled limit: %v", err)
	}
	if err := p.enforceWorkspaceLimit(sandboxPaths{}, 1024); err != nil {
		t.Fatalf("enforceWorkspaceLimit with zero paths: %v", err)
	}
}

func TestSandboxSlotSemaphore(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	p := New(db, Config{Sandbox: true, Limits: SandboxLimits{MaxSandboxes: 1}})

	id1, err := db.Add(store.Request{Kind: store.KindNew, Author: "alice", Body: "one"})
	if err != nil {
		t.Fatal(err)
	}
	id2, err := db.Add(store.Request{Kind: store.KindNew, Author: "bob", Body: "two"})
	if err != nil {
		t.Fatal(err)
	}
	it1, _ := db.Get(id1)
	it2, _ := db.Get(id2)

	release1 := p.acquireSandboxSlot(it1)
	defer release1() // always frees the channel slot (idempotent)
	if got := p.sandboxSlotsUsed(); got != 1 {
		t.Fatalf("slots in use = %d, want 1", got)
	}

	acquired := make(chan func(), 1)
	go func() { acquired <- p.acquireSandboxSlot(it2) }()
	select {
	case release := <-acquired:
		release()
		t.Fatal("second acquire succeeded while the only slot was held")
	case <-time.After(200 * time.Millisecond):
	}
	// While waiting, the request was re-marked queued with a reason.
	got, err := db.Get(id2)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.StatusQueued || !strings.Contains(got.Note, "sandbox slot") {
		t.Fatalf("waiting request = %q %q, want queued with a slot note", got.Status, got.Note)
	}

	release1()
	select {
	case release2 := <-acquired:
		release2()
	case <-time.After(2 * time.Second):
		t.Fatal("second acquire did not proceed after the slot was released")
	}
	// The release funcs are idempotent.
	release1()
	if got := p.sandboxSlotsUsed(); got != 0 {
		t.Fatalf("slots in use after release = %d, want 0", got)
	}
}

// TestStartWaitsForSandboxSlot: with MaxSandboxes=1 and the single slot
// occupied, start() of a request must not mark it working until a slot frees
// up. The acquire runs inside start (before the run goroutine), so it blocks
// the caller; the test therefore calls start in a goroutine.
func TestStartWaitsForSandboxSlot(t *testing.T) {
	db, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	root := t.TempDir()
	p := New(db, Config{
		Repo:            root, // unused: run() fails before touching it
		Sandbox:         true,
		SandboxRoot:     root,
		PreviewRoot:     root,
		Limits:          SandboxLimits{MaxSandboxes: 1},
		JanitorInterval: -1,
	})

	id1, err := db.Add(store.Request{Kind: store.KindNew, Author: "alice", Body: "one"})
	if err != nil {
		t.Fatal(err)
	}
	it1, _ := db.Get(id1)

	// Occupy the single slot: the request's start must queue behind it.
	// start() acquires the slot synchronously (before spawning the run
	// goroutine), so call it from a goroutine.
	hold := p.acquireSandboxSlot(store.Request{ID: 999, Kind: store.KindNew})
	defer hold() // always frees the channel slot (idempotent)
	t.Logf("hold acquired; slots %d/%d", p.sandboxSlotsUsed(), cap(p.slots))
	started := make(chan struct{})
	go func() { p.start(it1, ""); close(started) }()
	defer func() {
		// Wait for the run to finish failing so the temp DB outlives it.
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			p.mu.Lock()
			n := len(p.busy)
			p.mu.Unlock()
			if n == 0 {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		t.Error("the lane stayed busy after the slot was released")
	}()

	// While the slot is held the request must stay queued with a note.
	time.Sleep(300 * time.Millisecond)
	select {
	case <-started:
		t.Fatal("start returned while the only slot was held")
	default:
	}
	got, err := db.Get(id1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.StatusQueued || !strings.Contains(got.Note, "sandbox slot") {
		t.Fatalf("#%d = %q %q, want queued with a slot note while the slot is held",
			id1, got.Status, got.Note)
	}

	// Releasing the slot lets the request proceed; it fails fast in run()
	// (no agent configured) and frees the slot again.
	hold()
	select {
	case <-started:
	case <-time.After(10 * time.Second):
		t.Fatal("start() never returned after the slot was released")
	}
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		got, _ := db.Get(id1)
		if got.Status == store.StatusFailed {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	got, _ = db.Get(id1)
	if got.Status != store.StatusFailed {
		t.Fatalf("request did not run after the slot was released: #%d=%q", id1, got.Status)
	}
	if got := p.sandboxSlotsUsed(); got != 0 {
		t.Fatalf("slots in use after the run = %d, want 0", got)
	}
}

// TestSandboxRuntimeCapKillsServer starts a real sandbox whose command
// sleeps far beyond the runtime cap, stops it the way run() does on a
// waitForTurn timeout, and asserts the whole group (bwrap and the sleep
// inside it) is dead afterwards.
func TestSandboxRuntimeCapKillsServer(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping sandbox integration test in -short mode")
	}
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bwrap not available:", err)
	}
	root := t.TempDir()
	paths, err := sandboxPathsFor(root, 12, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	p := New(nil, Config{
		Sandbox:     true,
		SandboxRoot: root,
		Limits: SandboxLimits{
			RuntimeMax:   2 * time.Second, // the cap under test
			GracefulStop: 2 * time.Second,
		},
	})
	if err := p.createSandboxDirs(paths); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.Workspace, 0o755); err != nil {
		t.Fatal(err)
	}

	grace := p.cfg.Limits.GracefulStop
	handle, err := sandbox.Start(context.Background(), sandbox.Spec{
		Name:      "req-12",
		Workspace: paths.Workspace,
		Home:      paths.Home,
		State:     paths.State,
		Tmp:       paths.Tmp,
		GoCache:   paths.GoCache,
		ModCache:  "-",
		Command:   []string{"/bin/bash", "-c", "echo sandbox-ready; sleep 60"},
	})
	if err != nil {
		t.Fatalf("start the sandbox: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	for !strings.Contains(handle.OutputString(), "sandbox-ready") {
		if time.Now().After(deadline) {
			t.Fatalf("sandbox did not start; output: %s", handle.OutputString())
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Cleanup(func() { handle.Stop(0) })

	// The runtime cap fires: run() would stop the server through
	// stopShelley (grace, then the cgroup/pgroup kill inside Stop).
	start := time.Now()
	if err := handle.Stop(grace); err != nil {
		t.Fatalf("stop after the runtime cap: %v", err)
	}
	if elapsed := time.Since(start); elapsed > 15*time.Second {
		t.Fatalf("the stop took %s; the 60s sleep must die with the sandbox", elapsed)
	}
	if !handle.Exited() {
		t.Fatal("the sandbox process is still running after Stop")
	}

	// No process of the sandbox survives on the host: the recorded launcher
	// pid (systemd-run client or bwrap) is gone, and the transient systemd
	// scope is removed. The sleep(1) inside the PID namespace dies with its
	// bwrap parent (no grandchild survives: verified by the namespace
	// teardown, which SIGKILLs everything in the namespace).
	if processAlive(handle.PID) {
		t.Errorf("sandbox launcher pid %d is still alive", handle.PID)
	}
	if handle.Unit != "" {
		out, err := exec.Command("systemctl", "--user", "list-units", "--all", handle.Unit,
			"--no-legend", "--no-pager").Output()
		if err == nil && strings.Contains(string(out), handle.Unit) {
			t.Errorf("systemd unit %s survived: %s", handle.Unit, out)
		}
	}
}

// TestWaitForTurnDeadline verifies the runtime cap path of waitForTurn
// without a server: an unregistered sandbox target is skipped on every poll,
// so only the deadline can end the wait.
func TestWaitForTurnDeadline(t *testing.T) {
	p := New(nil, Config{Sandbox: true})
	paths, err := sandboxPathsFor(t.TempDir(), 13, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = p.waitForTurn(13, shelleyTarget{}, "conv-x", 150*time.Millisecond, paths)
	if err == nil {
		t.Fatal("waitForTurn returned nil after the deadline")
	}
	if !strings.Contains(err.Error(), "runtime limit") {
		t.Fatalf("waitForTurn error = %q, want a runtime-limit message", err)
	}
	if elapsed := time.Since(start); elapsed > 6*time.Second {
		t.Fatalf("waitForTurn took %s past a 150ms deadline", elapsed)
	}
}

// TestWaitForTurnWorkspaceLimitFailsFast plants an oversized sandbox before
// the first poll: the size check must fail the turn even though the deadline
// is far away.
func TestWaitForTurnWorkspaceLimitFailsFast(t *testing.T) {
	root := t.TempDir()
	paths, err := sandboxPathsFor(root, 14, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.State, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.State, "blob"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	p := New(nil, Config{
		Sandbox:     true,
		SandboxRoot: root,
		Limits:      SandboxLimits{WorkspaceMaxBytes: 128},
	})
	// A registered (fake) server so the sandbox poll path runs.
	p.registerServer(14, &sandboxServer{target: shelleyTarget{socket: paths.Socket}})
	defer p.unregisterServer(14)

	_, err = p.waitForTurn(14, shelleyTarget{}, "conv-x", time.Minute, paths)
	if err == nil {
		t.Fatal("waitForTurn returned nil for an oversized sandbox")
	}
	if !strings.Contains(err.Error(), "128 byte limit") {
		t.Fatalf("waitForTurn error = %q, want the workspace limit", err)
	}
}
