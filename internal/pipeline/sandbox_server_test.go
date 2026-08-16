package pipeline

import (
	"os"
	"strings"
	"testing"
)

func TestClientArgsLegacyTarget(t *testing.T) {
	args := clientArgs(shelleyTarget{}, "list", "-limit", "5")
	if got := strings.Join(args, " "); got != "list -limit 5" {
		t.Fatalf("legacy client args = %q", got)
	}
	for _, a := range args {
		if a == "-url" || a == "-H" {
			t.Fatalf("legacy target must not set -url/-H: %v", args)
		}
	}
}

// TestClientArgsSandboxTargetAlwaysSetsSocketAndHeader is the critical
// isolation property: a sandbox client command can never fall back to the
// default socket of the primary Shelley server.
func TestClientArgsSandboxTargetAlwaysSetsSocketAndHeader(t *testing.T) {
	target := shelleyTarget{socket: "/state/shelley.sock", header: "X-Sandbox-1a2b: deadbeef"}
	for _, sub := range [][]string{
		{"chat", "-cwd", "/w", "-p", "do it"},
		{"chat", "-c", "abc", "-p", "more"},
		{"list", "-limit", "200"},
		{"read", "abc"},
	} {
		args := clientArgs(target, sub...)
		joined := strings.Join(args, "\x00")
		if !strings.Contains(joined, "-url\x00unix:///state/shelley.sock") {
			t.Errorf("clientArgs(%v) missing -url unix:// socket: %v", sub, args)
		}
		if !strings.Contains(joined, "-H\x00X-Sandbox-1a2b: deadbeef") {
			t.Errorf("clientArgs(%v) missing -H header: %v", sub, args)
		}
		// The socket flags must come before the subcommand so the client's
		// global flag parsing sees them.
		if len(args) < 4 || args[0] != "-url" || args[2] != "-H" {
			t.Errorf("clientArgs(%v) must lead with -url/-H: %v", sub, args)
		}
	}
}

func TestGenerateHeaderName(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 100; i++ {
		name, err := generateHeaderName()
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasPrefix(name, "X-Sandbox-") {
			t.Fatalf("header name %q lacks X-Sandbox- prefix", name)
		}
		rest := strings.TrimPrefix(name, "X-Sandbox-")
		if len(rest) != 16 {
			t.Fatalf("header name %q: random part is %d chars, want 16", name, len(rest))
		}
		for _, r := range rest {
			if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
				t.Fatalf("header name %q contains non-hex %q", name, r)
			}
		}
		if seen[name] {
			t.Fatalf("header name %q generated twice", name)
		}
		seen[name] = true
	}
}

func TestDecideRecovery(t *testing.T) {
	integrityErr := errTest("clone broken")
	cases := []struct {
		name         string
		hasRun       bool
		convID       string
		integrityErr error
		want         recoveryAction
	}{
		{"fresh request", false, "", nil, recoveryFresh},
		{"fresh request, nothing to check", false, "", integrityErr, recoveryFresh},
		{"resume conversation with intact sandbox", true, "conv-1", nil, recoveryContinue},
		{"resume conversation with broken clones", true, "conv-1", integrityErr, recoveryFail},
		{"sandbox created but no conversation yet", true, "", nil, recoveryFresh},
		{"no conversation and broken clones", true, "", integrityErr, recoveryFail},
	}
	for _, tc := range cases {
		if got := decideRecovery(tc.hasRun, tc.convID, tc.integrityErr); got != tc.want {
			t.Errorf("%s: decideRecovery(%v, %q, %v) = %d, want %d",
				tc.name, tc.hasRun, tc.convID, tc.integrityErr, got, tc.want)
		}
	}
}

type errTest string

func (e errTest) Error() string { return string(e) }

// TestServerRegistry exercises the Pipeline's running-server bookkeeping:
// registration, target lookup, and the metadata pid/unit lifecycle around a
// stopped server (stopShelley on a nil handle must only clear metadata).
func TestServerRegistry(t *testing.T) {
	root := t.TempDir()
	paths, err := sandboxPathsFor(root, 42, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	if err := (&Pipeline{}).createSandboxDirs(paths); err != nil {
		t.Fatal(err)
	}
	meta := sandboxMetadata{
		RequestID: 42,
		Username:  "teacher",
		Version:   sandboxFormatVersion,
		Phase:     phaseAgentRunning,
		PID:       12345,
		Unit:      "learningmaterial-sandbox-req-42-deadbeef.scope",
	}
	if err := writeMetadata(paths, meta); err != nil {
		t.Fatal(err)
	}

	p := New(nil, Config{})
	target := shelleyTarget{socket: paths.Socket, header: "X-Sandbox-ab12: cd34"}
	srv := &sandboxServer{target: target, paths: paths, meta: meta}

	p.registerServer(42, srv)
	got, ok := p.serverTarget(42)
	if !ok || got != target {
		t.Fatalf("serverTarget after register = %v, %v; want %v, true", got, ok, target)
	}

	// stopShelley with no handle only clears the process bookkeeping.
	p.stopShelley(&sandboxServer{paths: paths, meta: meta})
	cleared, err := readMetadata(paths)
	if err != nil {
		t.Fatal(err)
	}
	if cleared.PID != 0 || cleared.Unit != "" {
		t.Fatalf("stopShelley left pid=%d unit=%q", cleared.PID, cleared.Unit)
	}
	if cleared.Phase != phaseAgentRunning {
		t.Fatalf("stopShelley changed phase to %q", cleared.Phase)
	}

	if unregistered := p.unregisterServer(42); unregistered != srv {
		t.Fatalf("unregisterServer returned %v, want the registered server", unregistered)
	}
	if _, ok := p.serverTarget(42); ok {
		t.Fatal("serverTarget still set after unregister")
	}
	if p.unregisterServer(42) != nil {
		t.Fatal("unregisterServer twice returned a server")
	}
}

// TestKillStaleSandboxNoState verifies recovery bookkeeping with no recorded
// process: nothing to kill, a stale socket file is removed, and no error
// surfaces from the absent unit/pid.
func TestKillStaleSandboxNoState(t *testing.T) {
	root := t.TempDir()
	paths, err := sandboxPathsFor(root, 7, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	if err := (&Pipeline{}).createSandboxDirs(paths); err != nil {
		t.Fatal(err)
	}
	if err := killStaleSandbox(sandboxMetadata{RequestID: 7}); err != nil {
		t.Fatal(err)
	}
	// Simulate the stale socket a crashed server left behind.
	if err := os.WriteFile(paths.Socket, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	staleSocketCleanup(paths)
	if _, err := os.Stat(paths.Socket); !os.IsNotExist(err) {
		t.Fatal("stale socket still present")
	}
}
