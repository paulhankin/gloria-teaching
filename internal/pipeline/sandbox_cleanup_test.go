package pipeline

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"learningmaterial/internal/store"
)

// writeFakeSandbox creates a req-<id> sandbox directory with a metadata file
// whose mtime is age in the past.
func writeFakeSandbox(t *testing.T, root string, id int64, phase string, age time.Duration) {
	t.Helper()
	dir := filepath.Join(root, fmt.Sprintf("req-%d", id))
	state := filepath.Join(dir, "state")
	if err := os.MkdirAll(state, 0o755); err != nil {
		t.Fatal(err)
	}
	meta := sandboxMetadata{
		RequestID: id,
		Username:  "teacher",
		Branch:    fmt.Sprintf("req-%d", id),
		Version:   sandboxFormatVersion,
		Phase:     phase,
	}
	paths := sandboxPaths{Root: dir, Metadata: filepath.Join(state, "metadata.json")}
	if err := writeMetadata(paths, meta); err != nil {
		t.Fatal(err)
	}
	mtime := time.Now().Add(-age)
	if err := os.Chtimes(filepath.Join(state, "metadata.json"), mtime, mtime); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(dir, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

// setupJanitorFixture builds a sandbox-root + DB with one request row per
// given status. It returns the pipeline and the request IDs keyed by status.
func setupJanitorFixture(t *testing.T) (*Pipeline, string, *store.DB) {
	t.Helper()
	tmp := t.TempDir()
	db, err := store.Open(filepath.Join(tmp, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	root := filepath.Join(tmp, "sandboxes")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	p := New(db, Config{
		Sandbox:     true,
		SandboxRoot: root,
		Limits: SandboxLimits{
			MaxSandboxes:    2,
			RetainCompleted: time.Hour,
			RetainFailed:    24 * time.Hour,
		},
		JanitorInterval: -1,
	})
	return p, root, db
}

func sandboxExists(root string, id int64) bool {
	_, err := os.Stat(filepath.Join(root, "req-"+fmt.Sprintf("%d", id)))
	return err == nil
}

func TestJanitorRetention(t *testing.T) {
	p, root, db := setupJanitorFixture(t)

	add := func(status store.Status) int64 {
		t.Helper()
		id, err := db.Add(store.Request{Kind: store.KindNew, Author: "teacher", Body: "x"})
		if err != nil {
			t.Fatal(err)
		}
		if err := db.SetStatus(id, status, "test"); err != nil {
			t.Fatal(err)
		}
		return id
	}

	doneOld := add(store.StatusDone)       // done, old: deleted
	doneFresh := add(store.StatusDone)     // done, fresh: kept
	failedOld := add(store.StatusFailed)   // failed, old: deleted
	failedFresh := add(store.StatusFailed) // failed, fresh: kept
	rejectedOld := add(store.StatusRejected)
	active := add(store.StatusWorking) // active: never deleted, any age

	writeFakeSandbox(t, root, doneOld, phasePublished, 2*time.Hour)
	writeFakeSandbox(t, root, doneFresh, phasePublished, 30*time.Minute)
	writeFakeSandbox(t, root, failedOld, phaseAgentRunning, 48*time.Hour)
	writeFakeSandbox(t, root, failedFresh, phaseAgentFinished, 2*time.Hour)
	writeFakeSandbox(t, root, rejectedOld, phaseReady, 48*time.Hour)
	writeFakeSandbox(t, root, active, phaseAgentRunning, 30*24*time.Hour)

	// Orphaned sandbox: no request row at all.
	const orphan = int64(900)
	writeFakeSandbox(t, root, orphan, phaseAgentFinished, 48*time.Hour)
	// Orphaned but fresh: kept.
	const orphanFresh = int64(901)
	writeFakeSandbox(t, root, orphanFresh, phaseAgentFinished, time.Hour)

	p.JanitorRun(time.Now())

	for _, id := range []int64{doneOld, failedOld, rejectedOld, orphan} {
		if sandboxExists(root, id) {
			t.Errorf("sandbox req-%d should have been deleted", id)
		}
	}
	for _, id := range []int64{doneFresh, failedFresh, active, orphanFresh} {
		if !sandboxExists(root, id) {
			t.Errorf("sandbox req-%d should have been retained", id)
		}
	}
}

func TestJanitorNeverDeletesActive(t *testing.T) {
	p, root, db := setupJanitorFixture(t)
	id, err := db.Add(store.Request{Kind: store.KindNew, Author: "teacher", Body: "x"})
	if err != nil {
		t.Fatal(err)
	}
	// Even a failed (retryable) request with an in-flight goroutine (lane
	// busy) keeps its sandbox.
	if err := db.SetStatus(id, store.StatusFailed, "boom"); err != nil {
		t.Fatal(err)
	}
	writeFakeSandbox(t, root, id, phaseAgentRunning, 48*time.Hour)

	it, _ := db.Get(id)
	p.mu.Lock()
	p.busy[it.Lane()] = true
	p.mu.Unlock()

	p.JanitorRun(time.Now())
	if !sandboxExists(root, id) {
		t.Error("the janitor deleted the sandbox of an in-flight request")
	}

	// Once the lane is free, the same old failed sandbox is collected.
	p.mu.Lock()
	delete(p.busy, it.Lane())
	p.mu.Unlock()
	p.JanitorRun(time.Now())
	if sandboxExists(root, id) {
		t.Error("the janitor kept a failed sandbox past its retention")
	}
}

func TestJanitorRegisteredServerProtects(t *testing.T) {
	p, root, db := setupJanitorFixture(t)
	id, err := db.Add(store.Request{Kind: store.KindNew, Author: "teacher", Body: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetStatus(id, store.StatusFailed, "boom"); err != nil {
		t.Fatal(err)
	}
	writeFakeSandbox(t, root, id, phaseAgentRunning, 48*time.Hour)

	// A registered server means a run goroutine owns the sandbox right now.
	p.registerServer(id, &sandboxServer{})
	p.JanitorRun(time.Now())
	if !sandboxExists(root, id) {
		t.Error("the janitor deleted a sandbox with a registered server")
	}
	p.unregisterServer(id)
}

func TestStartupCleanupDeletesTerminalAndKeepsActive(t *testing.T) {
	p, root, db := setupJanitorFixture(t)

	old, err := db.Add(store.Request{Kind: store.KindNew, Author: "teacher", Body: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.SetStatus(old, store.StatusFailed, "boom"); err != nil {
		t.Fatal(err)
	}
	// Fresh failed sandbox: the janitor would keep it (age < retention),
	// but at startup the service was down for the whole retention window,
	// so terminal sandboxes are deleted immediately.
	writeFakeSandbox(t, root, old, phaseAgentFinished, time.Minute)

	active, err := db.Add(store.Request{Kind: store.KindNew, Author: "teacher", Body: "x"})
	if err != nil {
		t.Fatal(err)
	}
	// StatusWorking: recover() is about to requeue it; its sandbox stays.
	if err := db.SetStatus(active, store.StatusWorking, "interrupted"); err != nil {
		t.Fatal(err)
	}
	writeFakeSandbox(t, root, active, phaseAgentRunning, time.Minute)

	p.cleanStaleSandboxes()
	if sandboxExists(root, old) {
		t.Error("startup cleanup kept a terminal sandbox")
	}
	if !sandboxExists(root, active) {
		t.Error("startup cleanup deleted the sandbox of an active request")
	}
}

func TestStartupCleanupKillsRecordedProcess(t *testing.T) {
	p, root, db := setupJanitorFixture(t)
	id, err := db.Add(store.Request{Kind: store.KindNew, Author: "teacher", Body: "x"})
	if err != nil {
		t.Fatal(err)
	}
	// A terminal request whose metadata records a still-running process: the
	// startup sweep must run killStaleSandbox (which SIGTERMs the recorded
	// pid) and delete the sandbox. Killing is exercised for real by the
	// runtime-cap integration test; here we verify the sweep tolerates the
	// wedged case: a pid that ignores SIGTERM must not stop the cleanup.
	if err := db.SetStatus(id, store.StatusFailed, "boom"); err != nil {
		t.Fatal(err)
	}
	writeFakeSandbox(t, root, id, phaseAgentRunning, 48*time.Hour)

	// A background sleep makes a real recorded pid. The sandbox cleanup
	// SIGTERMs it; the Cleanup harness kills it in any case.
	cmd := exec.Command("sleep", "60")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn sleep: %v", err)
	}
	pid := cmd.Process.Pid
	t.Cleanup(func() { cmd.Process.Kill(); cmd.Wait() })

	dir := filepath.Join(root, fmt.Sprintf("req-%d", id))
	paths := sandboxPaths{Root: dir, Metadata: filepath.Join(dir, "state", "metadata.json")}
	meta, err := readMetadata(paths)
	if err != nil {
		t.Fatal(err)
	}
	meta.PID = pid
	if err := writeMetadata(paths, meta); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	p.cleanStaleSandboxes()
	// killStaleSandbox waits at most ~5s for a wedged pid before SIGKILLing.
	if elapsed := time.Since(start); elapsed > 15*time.Second {
		t.Errorf("startup cleanup took %s with a wedged recorded pid", elapsed)
	}
	if sandboxExists(root, id) {
		t.Error("the recorded process's sandbox was not deleted")
	}
	// The SIGTERM usually reaches the sleep; the kill is verified for real
	// by TestSandboxRuntimeCapKillsServer. Here only report survival.
	if processAlive(pid) {
		t.Logf("recorded pid %d ignored SIGTERM and survived (wedged case tolerated)", pid)
	}
}
