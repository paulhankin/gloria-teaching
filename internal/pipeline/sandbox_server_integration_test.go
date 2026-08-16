package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"learningmaterial/internal/sandbox"
)

// Integration tests in this file launch a real "shelley serve" through the
// real bubblewrap launcher. They never start an LLM conversation (cost);
// list/read against an empty database covers the client routing.

func requireShelleySandbox(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping sandboxed shelley integration test in -short mode")
	}
	for _, tool := range []string{"shelley", "bwrap", "curl"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not available: %v", tool, err)
		}
	}
	if _, err := os.Stat("/exe.dev/shelley.json"); err != nil {
		t.Skip("/exe.dev/shelley.json not available:", err)
	}
}

// testSandboxLayout builds a minimal but complete sandbox layout: the
// directories createSandboxDirs makes plus a trivial (non-git) workspace.
//
// NOTE: it deliberately does not use t.TempDir(). The isolated server keeps
// running until Cleanup stops it, and Go removes the TempDir first (LIFO),
// which would make the server die with I/O errors and spam the log.
func testSandboxLayout(t *testing.T, id int64) sandboxPaths {
	t.Helper()
	root, err := os.MkdirTemp("", "sandbox-itest-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Logf("retaining failed sandbox at %s", root)
			return
		}
		os.RemoveAll(root)
	})
	paths, err := sandboxPathsFor(root, id, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	p := &Pipeline{}
	if err := p.createSandboxDirs(paths); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(paths.Workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(paths.Workspace, "MARKER.txt"), []byte("workspace ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	return paths
}

// startTestShelley starts the isolated server exactly like the pipeline does.
func startTestShelley(t *testing.T, paths sandboxPaths, meta sandboxMetadata) *sandboxServer {
	t.Helper()
	p := New(nil, Config{Sandbox: true, SandboxRoot: filepath.Dir(paths.Root)})
	srv, err := p.startShelley(paths, meta)
	if err != nil {
		t.Fatalf("startShelley: %v", err)
	}
	t.Cleanup(func() { p.stopShelley(srv) })
	return srv
}

func TestSandboxedShelleyServer(t *testing.T) {
	requireShelleySandbox(t)
	paths := testSandboxLayout(t, 9001)
	meta := sandboxMetadata{RequestID: 9001, Username: "teacher", Version: sandboxFormatVersion}
	srv := startTestShelley(t, paths, meta)

	// The socket appeared and the metadata recorded the process.
	if _, err := os.Stat(paths.Socket); err != nil {
		t.Fatalf("socket not created: %v", err)
	}
	stored, err := readMetadata(paths)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Phase != phaseAgentRunning || stored.PID <= 0 {
		t.Fatalf("metadata after start: phase=%q pid=%d unit=%q", stored.Phase, stored.PID, stored.Unit)
	}

	// list works against the isolated server (unix socket; the header is
	// sent but the socket path does not enforce it).
	out, err := exec.Command("shelley", append([]string{"client"}, clientArgs(srv.target, "list")...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("shelley client list via sandbox socket: %v: %s", err, out)
	}

	// The isolated server has its own database below state/: it answers list
	// with zero conversations regardless of what the primary Shelley holds
	// (the -db flag points at paths.DB, and the sandbox mounts only the
	// state directory).
	if strings.Contains(string(out), "conversation_id") {
		t.Errorf("fresh isolated server already has conversations: %s", out)
	}

	// The required header guards the TCP listener: without it the API
	// answers 403, with it 200. Read the actual port from the sandbox output
	// ("port" NNNNN in the "Server starting" log line).
	port := serverPort(t, srv.handle)
	probe := func(args ...string) string {
		cmd := exec.Command("curl", append([]string{"-s", "-o", "/dev/null", "-w", "%{http_code}"}, args...)...)
		code, err := cmd.Output()
		if err != nil {
			t.Fatalf("curl %v: %v", args, err)
		}
		return strings.TrimSpace(string(code))
	}
	if code := probe("http://localhost:" + port + "/api/conversations"); code != "403" {
		t.Errorf("TCP API without header: got %s, want 403", code)
	}
	headerName, _, _ := strings.Cut(srv.target.header, ": ")
	if code := probe("-H", srv.target.header, "http://localhost:"+port+"/api/conversations"); code != "200" {
		t.Errorf("TCP API with header %s: got %s, want 200", headerName, code)
	}

	// Inside the sandbox neither the primary Shelley socket nor the host
	// home directory is visible; the request socket and workspace are.
	targets := []struct {
		path string
		want string
	}{
		{paths.Socket, "request-socket-visible"},
		{filepath.Join(paths.Workspace, "MARKER.txt"), "workspace-visible"},
		{"/home/exedev/.config/shelley/shelley.sock", "primary-socket-hidden"},
		{"/home/exedev", "host-home-hidden"},
	}
	var script strings.Builder
	for _, tc := range targets {
		script.WriteString("if [ -e " + tc.path + " ]; then echo " + tc.want + "=yes; else echo " + tc.want + "=no; fi\n")
	}
	probeSpec := sandbox.Spec{
		Name:      "req-9001-probe",
		Workspace: paths.Workspace,
		Home:      paths.Home,
		State:     paths.State,
		Tmp:       paths.Tmp,
		GoCache:   paths.GoCache,
		Command:   []string{"/bin/bash", "-c", script.String()},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	probeHandle, err := sandbox.Start(ctx, probeSpec)
	if err != nil {
		t.Fatalf("start probe sandbox: %v", err)
	}
	if err := probeHandle.Wait(); err != nil {
		t.Fatalf("probe sandbox: %v\n%s", err, probeHandle.OutputString())
	}
	results := map[string]string{}
	for _, line := range strings.Split(probeHandle.OutputString(), "\n") {
		if k, v, ok := strings.Cut(line, "="); ok && results[k] == "" && (v == "yes" || v == "no") {
			results[k] = v
		}
	}
	for _, tc := range targets {
		want := "no"
		if tc.want == "request-socket-visible" || tc.want == "workspace-visible" {
			want = "yes"
		}
		if results[tc.want] != want {
			t.Errorf("%s = %q, want %q (path %s)", tc.want, results[tc.want], want, tc.path)
		}
	}

	// Graceful stop: the database survives intact and the metadata process
	// bookkeeping is cleared.
	p := New(nil, Config{Sandbox: true})
	p.stopShelley(srv)
	if _, err := os.Stat(paths.DB); err != nil {
		t.Errorf("database missing after graceful stop: %v", err)
	}
	stored, err = readMetadata(paths)
	if err != nil {
		t.Fatal(err)
	}
	if stored.PID != 0 || stored.Unit != "" {
		t.Errorf("stopShelley left pid=%d unit=%q in metadata", stored.PID, stored.Unit)
	}
	// The server must actually be gone: connecting to the socket fails.
	if _, err := exec.Command("shelley", "client", "-url", "unix://"+paths.Socket, "list").CombinedOutput(); err == nil {
		t.Error("shelley client list succeeded against a stopped server")
	}
}

// serverPort extracts the random TCP port of a sandboxed shelley from its
// captured output (the "Server starting" log line's port=NNNNN field).
func serverPort(t *testing.T, handle *sandbox.Handle) string {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		out := handle.OutputString()
		if i := strings.Index(out, "port="); i >= 0 {
			rest := out[i+len("port="):]
			var digits strings.Builder
			for _, r := range rest {
				if r < '0' || r > '9' {
					break
				}
				digits.WriteRune(r)
			}
			if digits.Len() > 0 {
				return digits.String()
			}
		}
		if handle.Exited() {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("no port found in server output:\n%s", handle.OutputString())
	return ""
}

// TestSandboxedShelleyRestartRecovery restarts the isolated server on the
// same database after a graceful stop and verifies that a CONVERSATION
// created before the restart can be continued after it (the plan's
// integration test 9). It also covers the startup hygiene: a second start
// on the same paths kills the stale process bookkeeping and removes a
// stale socket left behind by a crash.
//
// The conversation uses the builtin deterministic "predictable" model (the
// server runs with -predictable-only), so the turn is real end-to-end work
// through the sandboxed server (chat -> bash tool -> git commit in the
// workspace) without any LLM cost.
func TestSandboxedShelleyRestartRecovery(t *testing.T) {
	requireShelleySandbox(t)
	paths := testSandboxLayout(t, 9002)
	meta := sandboxMetadata{RequestID: 9002, Username: "teacher", Version: sandboxFormatVersion}

	p := New(nil, Config{Sandbox: true, ShelleyPredictableOnly: true})
	first := startTestShelleyPredictable(t, paths, meta)

	// Turn 1 before the restart: the predictable model runs the bash
	// command verbatim inside the sandbox and commits to the workspace.
	convID, err := p.chatNew(first.target, paths.Workspace,
		"bash: echo before-restart > progress.txt && git init -q -b main . && "+
			"git -c user.name=T -c user.email=t@t add progress.txt && "+
			"git -c user.name=T -c user.email=t@t commit -qm before-restart")
	if err != nil {
		t.Fatalf("chatNew: %v", err)
	}
	if convID == "" {
		t.Fatal("chatNew returned an empty conversation id")
	}
	waitConversationIdle(t, first.target, convID)
	if _, err := os.Stat(filepath.Join(paths.Workspace, "progress.txt")); err != nil {
		t.Fatalf("turn 1 did not write through the workspace bind: %v", err)
	}
	p.stopShelley(first)

	// Simulate what a service restart leaves behind: metadata with a dead
	// pid/unit and (sometimes) a stale socket file.
	stale, err := readMetadata(paths)
	if err != nil {
		t.Fatal(err)
	}
	stale.PID = first.handle.PID // already reaped
	stale.Unit = first.handle.Unit
	if err := writeMetadata(paths, stale); err != nil {
		t.Fatal(err)
	}
	if err := killStaleSandbox(stale); err != nil {
		t.Fatalf("killStaleSandbox: %v", err)
	}
	staleSocketCleanup(paths)
	// After a graceful stop the server normally removes its socket itself;
	// when it does not (crash), killStaleSandbox + staleSocketCleanup must.
	if _, err := os.Stat(paths.Socket); err == nil {
		t.Log("server removed its own socket on stop; simulating a crash-left socket instead")
		if err := os.WriteFile(paths.Socket, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		staleSocketCleanup(paths)
	}

	// Integrity passes and decides "continue" for a recorded conversation.
	if err := sandboxIntegrityPaths(paths); err != nil {
		t.Fatalf("integrity after graceful stop: %v", err)
	}
	if got := decideRecovery(true, convID, sandboxIntegrityPaths(paths)); got != recoveryContinue {
		t.Fatalf("decideRecovery = %d, want recoveryContinue", got)
	}

	second := startTestShelleyPredictable(t, paths, stale)
	if second.handle.PID == first.handle.PID {
		t.Fatal("restarted server reuses the pid of the first")
	}
	// The conversation SURVIVED the restart on the same database: list
	// still shows it, and the continuation turn (the pipeline's resume
	// prompt in production; here a deterministic second bash command)
	// runs against the same workspace and git history.
	out, err := exec.Command("shelley", append([]string{"client"}, clientArgs(second.target, "list")...)...).CombinedOutput()
	if err != nil {
		t.Fatalf("list against restarted server: %v: %s", err, out)
	}
	if !strings.Contains(string(out), convID) {
		t.Fatalf("conversation %s missing after the restart: %s", convID, out)
	}
	if err := p.chatContinue(second.target, convID,
		"bash: echo after-restart >> progress.txt && "+
			"git -c user.name=T -c user.email=t@t add progress.txt && "+
			"git -c user.name=T -c user.email=t@t commit -qm after-restart"); err != nil {
		t.Fatalf("chatContinue after restart: %v", err)
	}
	waitConversationIdle(t, second.target, convID)
	data, err := os.ReadFile(filepath.Join(paths.Workspace, "progress.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Fields(string(data)); fmt.Sprint(got) != "[before-restart after-restart]" {
		t.Fatalf("progress.txt = %q, want both turns applied", got)
	}
	if got := gitOK(t, paths.Workspace, "rev-list", "--count", "HEAD"); got != "2" {
		t.Fatalf("workspace commits after the restart = %s, want 2", got)
	}
	p.stopShelley(second)
}

// startTestShelleyPredictable starts the isolated server with the builtin
// predictable model as the only model (no LLM integration discovery).
func startTestShelleyPredictable(t *testing.T, paths sandboxPaths, meta sandboxMetadata) *sandboxServer {
	t.Helper()
	p := New(nil, Config{Sandbox: true, SandboxRoot: filepath.Dir(paths.Root), ShelleyPredictableOnly: true})
	srv, err := p.startShelley(paths, meta)
	if err != nil {
		t.Fatalf("startShelley: %v", err)
	}
	t.Cleanup(func() { p.stopShelley(srv) })
	return srv
}

// waitConversationIdle polls shelley client list until the conversation is
// no longer working (the turn ended) or the deadline passes.
func waitConversationIdle(t *testing.T, target shelleyTarget, convID string) {
	t.Helper()
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		out, err := exec.Command("shelley", append([]string{"client"}, clientArgs(target, "list", "-limit", "50")...)...).Output()
		if err == nil {
			for _, line := range strings.Split(string(out), "\n") {
				var c struct {
					ID      string `json:"conversation_id"`
					Working bool   `json:"working"`
				}
				if json.Unmarshal([]byte(line), &c) == nil && c.ID == convID && !c.Working {
					return
				}
			}
		}
		time.Sleep(250 * time.Millisecond)
	}
	t.Fatalf("conversation %s still working after the deadline", convID)
}

// sandboxIntegrityPaths wraps sandboxIntegrity for the layout used here,
// where the workspace is deliberately not a git repository: that check
// belongs to the pipeline's clones, so this test substitutes a db-only check.
func sandboxIntegrityPaths(paths sandboxPaths) error {
	if _, err := os.Stat(paths.DB); err != nil {
		return err
	}
	return nil
}
