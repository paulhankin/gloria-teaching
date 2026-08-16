package pipeline

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"learningmaterial/internal/sandbox"
)

// This file routes every Shelley client operation of one request through a
// dedicated, sandboxed "shelley serve" process instead of the primary Shelley
// server. See docs/stage-1-shelley-sandbox-plan.md, step 4.
//
// Threat-model note: the per-run required header only guards the server's
// random TCP listener (verified experimentally: shelley's require-header
// middleware is applied to the TCP listener only; the Unix socket listener
// skips both require-header and CSRF checks because it is considered local
// and trusted). Pipeline client calls go over the Unix socket, where the
// header is not needed. The header value is therefore a random name only; a
// "Name: Value" pair is rejected by the header lookup (the colon becomes part
// of the canonicalized name), so the installed shelley cannot express a
// secret VALUE at all.
//
// The header name is passed to shelley on the command line, which means it is
// visible in the host's process list while the server runs. That is accepted:
// it protects only the random TCP port and its disclosure gains an attacker
// nothing they could not already reach by guessing the port. The name is
// never written to the agent workspace or exported to the sandbox
// environment; the agent has no legitimate reason to contact the server's TCP
// port.

// shelleyTarget identifies the Shelley server one client operation talks to.
// The zero value is the legacy behavior: the default socket of the primary
// Shelley server and no extra header.
type shelleyTarget struct {
	socket string // request-local Unix socket; empty = default socket
	header string // "Name: Value" sent as -H; empty = no extra header
}

// clientArgs builds the full argument vector for one "shelley client"
// invocation. It is the ONLY place client arguments are constructed, so
// sandbox mode can never silently fall back to the default socket: when
// t.socket is set, -url unix://<socket> and -H <header> are always prepended.
func clientArgs(t shelleyTarget, args ...string) []string {
	var out []string
	if t.socket != "" {
		out = append(out, "-url", "unix://"+t.socket)
		if t.header != "" {
			out = append(out, "-H", t.header)
		}
	}
	return append(out, args...)
}

// generateHeaderName returns a random per-run header name,
// e.g. "X-Sandbox-9f2ab41c0d7e3b65". The name itself carries the entropy;
// shelley's require-header checks the header's presence, not its value.
func generateHeaderName() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate sandbox header name: %w", err)
	}
	return "X-Sandbox-" + hex.EncodeToString(buf), nil
}

// sandboxServer is one running isolated Shelley server.
type sandboxServer struct {
	handle   *sandbox.Handle
	target   shelleyTarget // socket + header every client call of this request uses
	paths    sandboxPaths  // for metadata pid/unit cleanup
	meta     sandboxMetadata
	flushLog func() // drains the server.log tee after Stop; may be nil
}

// teeServeLog copies everything the sandbox writes to stdout/stderr into
// server.log until the sandbox exits. It returns a flush func that waits for
// the final write; the pipeline calls it after Stop.
func teeServeLog(handle *sandbox.Handle, path string) (flush func()) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return func() {}
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer f.Close()
		seen := 0
		for {
			out := handle.OutputString()
			if len(out) > seen {
				f.WriteString(out[seen:])
				seen = len(out)
			}
			if handle.Exited() {
				// One final drain: the process can have written output between
				// the last poll and reaping.
				if out := handle.OutputString(); len(out) > seen {
					f.WriteString(out[seen:])
				}
				return
			}
			time.Sleep(200 * time.Millisecond)
		}
	}()
	return func() { <-done }
}

// startShelley launches the isolated "shelley serve" of one request inside
// the bubblewrap sandbox and waits for its Unix socket. On success the
// metadata records pid/unit and the agent-running phase, and the returned
// server holds the client target (socket + required header) for every
// subsequent shelley client call of this request.
//
// The server runs with its own database, socket, and a random TCP port
// (-port 0). There is deliberately no shelley flag for a tool allowlist or
// for disabling JIT package installation (verified against shelley 0.930):
// the read-only system root makes installation attempts fail harmlessly, and
// tool restriction is left to the prompt (see chatExtraArgs).
func (p *Pipeline) startShelley(paths sandboxPaths, meta sandboxMetadata) (*sandboxServer, error) {
	if err := killStaleSandbox(meta); err != nil {
		return nil, err
	}
	staleSocketCleanup(paths)

	headerName, err := generateHeaderName()
	if err != nil {
		return nil, err
	}
	header := headerName + ": " + randomHexToken(8)

	command := []string{
		"shelley",
		"-db", paths.DB,
		"-config", "/exe.dev/shelley.json",
	}
	if p.cfg.ShelleyPredictableOnly {
		// Tests only: the builtin deterministic "predictable" model is the
		// only model the server offers (no LLM integration discovery).
		command = append(command, "-predictable-only")
	}
	command = append(command,
		"serve",
		"-socket", paths.Socket,
		"-port", "0",
		"-require-header", headerName,
	)

	spec := sandbox.Spec{
		Name:      fmt.Sprintf("req-%d", meta.RequestID),
		Workspace: paths.Workspace,
		Home:      paths.Home,
		State:     paths.State,
		Tmp:       paths.Tmp,
		GoCache:   paths.GoCache,
		Limits: sandbox.Limits{
			MemoryMax: p.cfg.Limits.MemoryMax,
			TasksMax:  p.cfg.Limits.TasksMax,
			CPUQuota:  p.cfg.Limits.CPUQuota,
		},
		Command: command,
	}
	if err := sandbox.SelfCheck(spec); err != nil {
		return nil, fmt.Errorf("sandbox self-check: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	handle, err := sandbox.Start(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("start sandboxed shelley: %w", err)
	}
	flushLog := teeServeLog(handle, paths.ServerLog)

	if err := sandbox.WaitSocket(ctx, paths.Socket, 30*time.Second); err != nil {
		handle.Stop(p.cfg.Limits.GracefulStop)
		flushLog()
		return nil, fmt.Errorf("sandboxed shelley did not become ready: %w\n%s",
			err, tail(handle.OutputString(), 800))
	}

	meta.PID = handle.PID
	meta.Unit = handle.Unit
	meta.Phase = phaseAgentRunning
	if err := writeMetadata(paths, meta); err != nil {
		handle.Stop(p.cfg.Limits.GracefulStop)
		flushLog()
		return nil, err
	}

	return &sandboxServer{
		handle:   handle,
		target:   shelleyTarget{socket: paths.Socket, header: header},
		paths:    paths,
		meta:     meta,
		flushLog: flushLog,
	}, nil
}

// stopShelley gracefully stops the request's isolated server, flushes its
// log, and clears the recorded pid/unit from the metadata. It is tolerant of
// a partially started server (nil fields).
func (p *Pipeline) stopShelley(srv *sandboxServer) {
	if srv == nil {
		return
	}
	if srv.handle != nil {
		if err := srv.handle.Stop(p.cfg.Limits.GracefulStop); err != nil {
			p.Logf("sandbox %s stop: %v", srv.handle.Name, err)
		}
	}
	if srv.flushLog != nil {
		srv.flushLog()
	}
	if latest, err := readMetadata(srv.paths); err == nil {
		latest.PID = 0
		latest.Unit = ""
		if err := writeMetadata(srv.paths, latest); err != nil {
			p.Logf("#%d: clear sandbox pid/unit: %v", srv.meta.RequestID, err)
		}
	}
}

// killStaleSandbox terminates a server recorded in metadata by a previous
// service run. It must run before touching the sandbox database or socket.
// A stale socket file is removed only after no live process owns it.
func killStaleSandbox(meta sandboxMetadata) error {
	if meta.Unit != "" {
		// Only ever stop units this service created; metadata is trusted,
		// but a corrupted file must not turn into a systemctl call against
		// an arbitrary unit.
		if strings.HasPrefix(meta.Unit, "learningmaterial-sandbox-") {
			// Stopping the scope SIGTERMs the whole cgroup. Errors are
			// ignored: the unit usually no longer exists.
			exec.Command("systemctl", "--user", "stop", meta.Unit).Run()
		}
	}
	if meta.PID > 0 && processAlive(meta.PID) {
		// The recorded pid is the host-side launcher (systemd-run or bwrap);
		// killing it tears down the sandbox (--die-with-parent, scope kill).
		syscall.Kill(meta.PID, syscall.SIGTERM)
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) && processAlive(meta.PID) {
			time.Sleep(100 * time.Millisecond)
		}
		if processAlive(meta.PID) {
			syscall.Kill(meta.PID, syscall.SIGKILL)
		}
	}
	return nil
}

// processAlive reports whether a host process with the given pid exists.
func processAlive(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}

// randomHexToken returns n random hex characters. It backs the required
// header VALUE; shelley only checks the name, but sending a random value
// keeps the wire format a normal "Name: Value" header.
func randomHexToken(n int) string {
	buf := make([]byte, (n+1)/2)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%x", time.Now().UnixNano())[:n]
	}
	return hex.EncodeToString(buf)[:n]
}

// chatExtraArgs are appended to every "shelley client chat" invocation of a
// sandboxed request. Config.ChatExtraArgs (tests/development only) comes
// first so a future conversation option here always wins over it.
//
// TODO(stage-1): restrict the conversation to the worksheet tool allowlist
// (bash, patch, keyword_search, change_dir). shelley 0.930 has neither a CLI
// flag nor a conversation-creation API option for a tool allowlist, and no
// flag to disable JIT package installation; the plumbing point is here so a
// future shelley version only needs to add arguments in one place. Until
// then the allowlist is expressed in the prompt only.
func (p *Pipeline) chatExtraArgs() []string {
	return p.cfg.ChatExtraArgs
}

// shelleyClient runs "shelley client <args...>" against target.
func (p *Pipeline) shelleyClient(t shelleyTarget, args ...string) *exec.Cmd {
	return exec.Command("shelley", append([]string{"client"}, clientArgs(t, args...)...)...)
}

// staleSocketCleanup removes a request socket file that no server owns. It
// must only be called after killStaleSandbox.
func staleSocketCleanup(paths sandboxPaths) {
	if _, err := os.Stat(paths.Socket); err == nil {
		os.Remove(paths.Socket)
	}
}

// sandboxIntegrity re-validates a surviving sandbox before it is reused: both
// clones must still be standalone repositories inside the request directory
// and the conversation database must exist. It returns an actionable error
// describing what recovery is impossible.
func sandboxIntegrity(paths sandboxPaths) error {
	if err := validateClone(paths.Workspace); err != nil {
		return fmt.Errorf("core clone failed integrity check: %w", err)
	}
	if err := validateClone(paths.WorkspaceUserRepo); err != nil {
		return fmt.Errorf("worksheet clone failed integrity check: %w", err)
	}
	if _, err := os.Stat(paths.DB); err != nil {
		return fmt.Errorf("isolated Shelley database %s: %w", paths.DB, err)
	}
	return nil
}

// recoveryAction decides what the sandbox path of run() does with a request
// that already has recorded state. It is a pure function so the recovery
// matrix is unit-testable.
type recoveryAction int

const (
	recoveryFresh    recoveryAction = iota // no usable state: set up from scratch
	recoveryContinue                       // restart the server and continue the stored conversation
	recoveryFail                           // state exists but is unusable: fail with an actionable error
)

// decideRecovery maps sandbox state to a recoveryAction. hasRun is whether
// the request record already points at a sandbox workspace; convID is the
// recorded conversation; integrityErr is the result of sandboxIntegrity.
func decideRecovery(hasRun bool, convID string, integrityErr error) recoveryAction {
	if !hasRun {
		return recoveryFresh
	}
	if convID == "" {
		// The sandbox was created but no conversation ever started (the
		// service died between clone creation and chatNew). The clones are
		// pristine; a fresh conversation can simply reuse them.
		if integrityErr == nil {
			return recoveryFresh
		}
		return recoveryFail
	}
	if integrityErr != nil {
		return recoveryFail
	}
	return recoveryContinue
}
