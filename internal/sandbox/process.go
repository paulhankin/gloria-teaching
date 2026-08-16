// Process lifecycle for sandbox launches: systemd scope when the user
// manager works, process-group fallback otherwise.
package sandbox

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Handle is a running sandbox. Exactly one of Unit (systemd scope launch) or
// pgid (fallback launch) identifies the process group for Stop. The Unit is
// discovered from the "Running as unit: X.scope" line systemd-run prints
// before entering the scope, which is why Start always wires the child's
// stdout/stderr to pipes: any caller-set cmd.Stdout would hide that line.
type Handle struct {
	Name string // sandbox name (Spec.Name)
	Unit string // transient systemd scope name, empty in fallback mode
	PID  int    // process ID of the direct child (systemd-run or bwrap)

	cmd     *exec.Cmd // the direct child process
	pgid    int       // fallback mode: process group ID, 0 in systemd mode
	waitCh  chan error
	waitErr error
	waitMu  sync.Mutex
	stopMu  sync.Mutex
	stopped bool

	// Output receives the sandbox's combined stdout/stderr as line chunks.
	// Read it with OutputString or copy from it; it is unclosed until the
	// process exits, so range over it to consume until exit.
	Output *bytes.Buffer
	outMu  sync.Mutex
}

// OutputString returns everything written to the sandbox's stdout/stderr so
// far. It is safe to call while the sandbox runs and after Wait.
func (h *Handle) OutputString() string {
	h.outMu.Lock()
	defer h.outMu.Unlock()
	return h.Output.String()
}

// String renders a short identifier for logs.
func (h *Handle) String() string {
	if h.Unit != "" {
		return fmt.Sprintf("%s (unit %s, pid %d)", h.Name, h.Unit, h.PID)
	}
	return fmt.Sprintf("%s (pgid %d, pid %d)", h.Name, h.pgid, h.PID)
}

// Wait blocks until the sandbox process exits and returns its exit error
// (nil on a clean exit 0).
func (h *Handle) Wait() error {
	<-h.waitCh
	h.waitMu.Lock()
	defer h.waitMu.Unlock()
	return h.waitErr
}

// waitClosed reports whether the wait channel has been closed (process
// reaped).
func (h *Handle) waitClosed() bool {
	select {
	case _, ok := <-h.waitCh:
		return !ok
	default:
		return false
	}
}

// Stop terminates the whole sandbox: first a graceful SIGTERM to the scope
// (systemd mode) or process group (fallback mode), then, after grace, a hard
// kill of the cgroup/process group. Stop is idempotent and safe to call
// after the sandbox already exited.
func (h *Handle) Stop(grace time.Duration) error {
	h.stopMu.Lock()
	defer h.stopMu.Unlock()
	if h.stopped {
		return nil
	}
	h.stopped = true

	if grace <= 0 {
		grace = 10 * time.Second
	}

	// Graceful phase.
	if h.Unit != "" {
		// Ask systemd to stop the scope: it SIGTERMs every process in the
		// cgroup and removes the scope.
		ctx, cancel := context.WithTimeout(context.Background(), grace)
		_ = exec.CommandContext(ctx, "systemctl", "--user", "stop", h.Unit).Run()
		cancel()
	} else if h.pgid > 0 {
		_ = syscall.Kill(-h.pgid, syscall.SIGTERM)
	}

	deadline := time.Now().Add(grace)
	for time.Now().Before(deadline) {
		if h.waitClosed() {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Forced phase.
	if h.Unit != "" {
		_ = exec.Command("systemctl", "--user", "kill", "--signal=KILL", h.Unit).Run()
		// The SIGKILL above goes to the cgroup; make sure the direct child
		// (the systemd-run client) is gone too. --collect normally reaps the
		// scope when the last process dies.
		if h.cmd != nil && h.cmd.Process != nil {
			_ = h.cmd.Process.Kill()
		}
	} else if h.pgid > 0 {
		_ = syscall.Kill(-h.pgid, syscall.SIGKILL)
		if h.cmd != nil && h.cmd.Process != nil {
			_ = h.cmd.Process.Kill()
		}
	}

	// Wait briefly for the child to be reaped so callers can rely on Wait
	// returning promptly after Stop.
	deadline = time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if h.waitClosed() {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("sandbox %s did not exit after kill", h.Name)
}

// --- systemd-run probe -----------------------------------------------------

// launchMode selects how sandboxes are wrapped for resource control.
type launchMode int

const (
	modeUnknown launchMode = iota
	modeSystemd            // systemd-run --user --scope works; use it
	modePgid               // fallback: plain process group with Setpgid
)

var (
	probeOnce sync.Once
	probeMode launchMode
	probeErr  error
)

// probeLaunchMode determines once whether "systemd-run --user --scope" works
// for this user. On VMs without a user manager (no logind session, container
// without systemd user instances) every systemd-run call would fail, so the
// fallback launches bwrap directly in a new process group and Stop kills the
// group with kill(-pgid). The result is cached for the process lifetime.
func probeLaunchMode() (launchMode, error) {
	probeOnce.Do(func() {
		if _, err := exec.LookPath("systemd-run"); err != nil {
			probeMode, probeErr = modePgid, fmt.Errorf("systemd-run not found: %w", err)
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "systemd-run", "--user", "--scope", "--quiet", "true")
		if out, err := cmd.CombinedOutput(); err != nil {
			probeMode = modePgid
			probeErr = fmt.Errorf("systemd-run --user probe failed: %v: %s", err, strings.TrimSpace(string(out)))
			return
		}
		probeMode = modeSystemd
	})
	return probeMode, probeErr
}

// LaunchMode reports the probed launch mode for diagnostics: "systemd" or
// "pgid". It runs the probe on first use.
func LaunchMode() string {
	mode, _ := probeLaunchMode()
	if mode == modeSystemd {
		return "systemd"
	}
	return "pgid"
}

// randomSuffix returns n random hex characters for unit names.
func randomSuffix(n int) string {
	buf := make([]byte, (n+1)/2)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand failure is vanishingly unlikely; fall back to time.
		return fmt.Sprintf("%x", time.Now().UnixNano())[:n]
	}
	return hex.EncodeToString(buf)[:n]
}

// Start launches spec inside bubblewrap. When the systemd user manager is
// available the sandbox runs in a transient scope (resource limits as scope
// properties, whole-cgroup kill via systemctl); otherwise it runs in a new
// process group (Stop kills the group). The returned Handle tracks the
// direct child; Wait reports its exit.
func Start(ctx context.Context, spec Spec) (*Handle, error) {
	bwrap, err := bwrapPath()
	if err != nil {
		return nil, err
	}
	args, err := Args(spec)
	if err != nil {
		return nil, err
	}

	mode, probeErr := probeLaunchMode()

	h := &Handle{Name: spec.Name, waitCh: make(chan error, 1), Output: &bytes.Buffer{}}
	var cmd *exec.Cmd

	if mode == modeSystemd {
		unit := unitPrefix + spec.Name + "-" + randomSuffix(8)
		sysArgs := []string{
			"--user", "--scope", "--collect",
			"--unit=" + unit,
		}
		if spec.Limits.MemoryMax != "" {
			sysArgs = append(sysArgs, "-p", "MemoryMax="+spec.Limits.MemoryMax)
		}
		if spec.Limits.TasksMax > 0 {
			sysArgs = append(sysArgs, "-p", fmt.Sprintf("TasksMax=%d", spec.Limits.TasksMax))
		}
		if spec.Limits.CPUQuota != "" {
			sysArgs = append(sysArgs, "-p", "CPUQuota="+spec.Limits.CPUQuota)
		}
		sysArgs = append(sysArgs, "--", bwrap)
		sysArgs = append(sysArgs, args...)

		cmd = exec.CommandContext(ctx, "systemd-run", sysArgs...)
		h.Unit = unit
	} else {
		// No working user manager: run bwrap directly in its own process
		// group. bwrap's --die-with-parent plus Stop's kill(-pgid) provide
		// the group lifecycle. Resource limits are not applied in this
		// mode; log-worthy, the caller can surface probeErr.
		cmd = exec.CommandContext(ctx, bwrap, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	}

	// Capture stdout/stderr: systemd-run announces the scope name on its
	// stderr and then relays the sandbox output; in fallback mode the pipes
	// carry bwrap output directly.
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		if mode == modeSystemd && probeErr == nil {
			return nil, fmt.Errorf("start systemd-run for sandbox %s: %w", spec.Name, err)
		}
		return nil, fmt.Errorf("start bwrap for sandbox %s: %w", spec.Name, err)
	}
	h.cmd = cmd
	h.PID = cmd.Process.Pid
	if mode == modePgid {
		h.pgid = cmd.Process.Pid // Setpgid makes the child its own group leader
	}

	// Pump both pipes into h.Output until the process exits.
	var pump sync.WaitGroup
	pump.Add(2)
	for _, r := range []io.Reader{stdout, stderr} {
		go func(r io.Reader) {
			defer pump.Done()
			buf := make([]byte, 32*1024)
			for {
				n, err := r.Read(buf)
				if n > 0 {
					h.outMu.Lock()
					h.Output.Write(buf[:n])
					h.outMu.Unlock()
				}
				if err != nil {
					return
				}
			}
		}(r)
	}

	go func() {
		err := cmd.Wait()
		pump.Wait()
		h.waitMu.Lock()
		h.waitErr = err
		h.waitMu.Unlock()
		h.waitCh <- err
		close(h.waitCh)
	}()

	if mode == modeSystemd {
		// systemd-run announces the scope before entering it; the line's
		// appearance is the readiness signal that the scope exists and the
		// sandbox process is about to run.
		needle := "Running as unit: " + h.Unit
		deadline := time.Now().Add(30 * time.Second)
		for {
			if strings.Contains(h.OutputString(), needle) {
				return h, nil
			}
			select {
			case <-ctx.Done():
				_ = h.Stop(0)
				return nil, ctx.Err()
			case err := <-h.waitCh:
				h.waitMu.Lock()
				h.waitErr = err
				h.waitMu.Unlock()
				return nil, fmt.Errorf("sandbox %s exited during launch: %v\n%s", spec.Name, err, strings.TrimSpace(h.OutputString()))
			default:
			}
			if time.Now().After(deadline) {
				_ = h.Stop(0)
				return nil, fmt.Errorf("sandbox %s: timed out waiting for systemd scope %s", spec.Name, h.Unit)
			}
			time.Sleep(20 * time.Millisecond)
		}
	}

	// Fallback mode: detect immediate bwrap argument/mount failures so
	// callers get a synchronous error instead of a silently dead sandbox.
	select {
	case <-ctx.Done():
		_ = h.Stop(0)
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		return h, nil
	case err := <-h.waitCh:
		h.waitMu.Lock()
		h.waitErr = err
		h.waitMu.Unlock()
		if probeErr != nil {
			return nil, fmt.Errorf("sandbox %s exited during launch (%v); systemd user manager unavailable: %v\n%s", spec.Name, err, probeErr, strings.TrimSpace(h.OutputString()))
		}
		return nil, fmt.Errorf("sandbox %s exited during launch: %v\n%s", spec.Name, err, strings.TrimSpace(h.OutputString()))
	}
}

// WaitSocket reports readiness of a Unix socket: the file must exist AND a
// connection attempt must succeed. It polls until the deadline or ctx
// cancellation and returns the last connection error.
func WaitSocket(ctx context.Context, path string, timeout time.Duration) error {
	if path == "" {
		return errors.New("socket path is empty")
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for {
		if _, err := os.Stat(path); err == nil {
			d := net.Dialer{Timeout: 500 * time.Millisecond}
			conn, err := d.DialContext(ctx, "unix", path)
			if err == nil {
				conn.Close()
				return nil
			}
			lastErr = err
		} else {
			lastErr = err
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("socket %s not ready after %s: %v", path, timeout, lastErr)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}
