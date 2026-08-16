package sandbox

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testSpec returns a valid Spec rooted in t.TempDir(). The Go build cache
// lives below the synthetic home, mirroring the pipeline layout.
func testSpec(t *testing.T) Spec {
	t.Helper()
	root := t.TempDir()
	spec := Spec{
		Name:      "test-" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-")),
		Workspace: filepath.Join(root, "workspace"),
		Home:      filepath.Join(root, "home"),
		State:     filepath.Join(root, "state"),
		Tmp:       filepath.Join(root, "tmp"),
		ModCache:  "-", // tests must not depend on the host module cache
		Command:   []string{"/bin/bash", "-c", "true"},
	}
	spec.GoCache = filepath.Join(spec.Home, ".cache", "go-build")
	for _, dir := range []string{spec.Workspace, spec.Home, spec.State, spec.Tmp, spec.GoCache} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return spec
}

// --- unit tests ------------------------------------------------------------

func TestArgsNamespaceAndProcessOptions(t *testing.T) {
	args, err := Args(testSpec(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, flag := range []string{
		"--unshare-user",
		"--unshare-pid",
		"--unshare-ipc",
		"--unshare-uts",
		"--unshare-cgroup-try",
		"--disable-userns",
		"--new-session",
		"--die-with-parent",
	} {
		if !argPresent(args, flag) {
			t.Errorf("args missing %s", flag)
		}
	}
	if !argsContainPair(args, "--cap-drop", "ALL") {
		t.Error("args missing --cap-drop ALL")
	}
	if !argsContainPair(args, "--proc", "/proc") {
		t.Error("args missing --proc /proc")
	}
	if !argsContainPair(args, "--dev", "/dev") {
		t.Error("args missing --dev /dev")
	}
	// Stage 1 must NOT unshare the network namespace, and --unshare-all
	// would include it.
	for _, a := range args {
		if strings.HasPrefix(a, "--unshare-net") || a == "--unshare-all" {
			t.Errorf("network isolation must stay disabled in stage 1, found %s", a)
		}
	}
}

func argPresent(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func TestArgsNeverMountForbiddenPaths(t *testing.T) {
	spec := testSpec(t)
	args, err := Args(spec)
	if err != nil {
		t.Fatal(err)
	}

	if argsContainPair(args, "--ro-bind", "/") || argsContainPair(args, "--bind", "/") {
		t.Error("args bind the host root")
	}
	// The real home directory must never be a bind source.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	realHome, err := filepath.EvalSymlinks(home)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i+1 < len(args); i++ {
		if args[i] != "--bind" && args[i] != "--ro-bind" {
			continue
		}
		src := args[i+1]
		if src == "/etc" {
			t.Error("args bind all of /etc")
		}
		if src == realHome || strings.HasPrefix(src, realHome+string(filepath.Separator)) {
			t.Errorf("args expose the real home: %s %s", args[i], src)
		}
		for _, never := range []string{"/users", "/var/lib", "/root"} {
			if src == never || strings.HasPrefix(src, never+string(filepath.Separator)) {
				t.Errorf("args expose forbidden path: %s %s", args[i], src)
			}
		}
	}
	// No host runtime sockets.
	joined := strings.Join(args, " ")
	for _, sock := range []string{"/run/user", "dbus", "docker.sock", "ssh-agent", "SSH_AUTH_SOCK"} {
		if strings.Contains(joined, sock) {
			t.Errorf("args mention forbidden socket path %q", sock)
		}
	}
}

func TestArgsEnvironmentPolicy(t *testing.T) {
	spec := testSpec(t)
	spec.Env = []string{"EXEDEV=1", "CI=true"}
	args, err := Args(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !argPresent(args, "--clearenv") {
		t.Fatal("args must start the environment from --clearenv")
	}
	setenv := map[string]string{}
	for i := 0; i+2 < len(args); i++ {
		if args[i] == "--setenv" {
			setenv[args[i+1]] = args[i+2]
		}
	}
	want := map[string]string{
		"HOME":       "/home/shelley",
		"GOCACHE":    "/home/shelley/.cache/go-build",
		"GOPATH":     "/home/shelley/go",
		"LANG":       "C.UTF-8",
		"LC_ALL":     "C.UTF-8",
		"TMPDIR":     "/tmp",
		"EXEDEV":     "1",
		"CI":         "true",
		"GOMODCACHE": "/home/shelley/go/pkg/mod", // ModCache "-": sandbox-local default
	}
	for k, v := range want {
		if setenv[k] != v {
			t.Errorf("setenv %s = %q, want %q", k, setenv[k], v)
		}
	}
	for _, dir := range strings.Split(setenv["PATH"], ":") {
		switch dir {
		case "/usr/local/go/bin", "/usr/local/bin", "/usr/bin", "/bin":
		default:
			t.Errorf("sandbox PATH contains unexpected dir %q", dir)
		}
	}
	// Sensitive names in the fixed set must never appear as values of other
	// variables either; spot check that nothing hosts a secret-shaped name.
	for name := range setenv {
		if err := checkExtraEnv(name + "=x"); err != nil && strings.HasPrefix(name, "EXEDEV") {
			t.Errorf("fixed env unexpectedly rejected: %v", err)
		}
	}
}

func TestArgsRejectsSensitiveEnv(t *testing.T) {
	for _, name := range []string{
		"AWS_SECRET_ACCESS_KEY",
		"ANTHROPIC_API_KEY",
		"SSH_AUTH_SOCK",
		"SHELLEY_URL",
		"HTTP_PROXY",
		"GIT_ASKPASS",
		"GITHUB_TOKEN",
		"DB_PASSWORD",
		"CLIENT_CREDENTIALS",
		"HOME", // fixed by policy, not overrideable
		"GOCACHE",
		"not-a-key-value",
	} {
		spec := testSpec(t)
		spec.Env = []string{name + "=value"}
		if _, err := Args(spec); err == nil {
			t.Errorf("Args accepted sensitive environment entry %q", name)
		}
	}
}

func TestArgsSkipsNonexistentMountSources(t *testing.T) {
	spec := testSpec(t)
	spec.ModCache = filepath.Join(t.TempDir(), "no-such-modcache")
	spec.ShelleyPath = filepath.Join(t.TempDir(), "no-such-shelley")
	args, err := Args(spec)
	if err != nil {
		t.Fatalf("missing mount sources must not fail Args: %v", err)
	}
	joined := strings.Join(args, " ")
	if strings.Contains(joined, "no-such-") {
		t.Error("args contain a nonexistent mount source")
	}
	// GOMODCACHE falls back to the sandbox-local default when the mount is
	// disabled or missing.
	if !argsContainPair(args, "GOMODCACHE", "/home/shelley/go/pkg/mod") {
		t.Error("GOMODCACHE does not point at the sandbox-local default")
	}
}

func TestArgsGoCacheOutsideHomeIsBindMounted(t *testing.T) {
	spec := testSpec(t)
	spec.GoCache = filepath.Join(t.TempDir(), "gocache")
	if err := os.MkdirAll(spec.GoCache, 0o755); err != nil {
		t.Fatal(err)
	}
	args, err := Args(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !argsContainPair(args, "--bind", filepath.Clean(spec.GoCache)) {
		t.Error("external GoCache is not bind-mounted")
	}
	v, ok := argAfter(args, "GOCACHE")
	if !ok || v != filepath.Clean(spec.GoCache) {
		t.Errorf("GOCACHE = %q, want %q", v, spec.GoCache)
	}
}

func TestArgsValidation(t *testing.T) {
	bad := func(mutate func(*Spec)) Spec {
		spec := testSpec(t)
		mutate(&spec)
		return spec
	}
	cases := map[string]Spec{
		"empty name":         bad(func(s *Spec) { s.Name = "" }),
		"name with slash":    bad(func(s *Spec) { s.Name = "../evil" }),
		"relative workspace": bad(func(s *Spec) { s.Workspace = "rel" }),
		"missing command":    bad(func(s *Spec) { s.Command = nil }),
		"empty home":         bad(func(s *Spec) { s.Home = "" }),
	}
	for name, spec := range cases {
		if _, err := Args(spec); err == nil {
			t.Errorf("%s: Args accepted an invalid spec", name)
		}
	}
}

func TestArgsDefaultModCacheIsMountedReadOnly(t *testing.T) {
	spec := testSpec(t)
	spec.ModCache = "" // resolve the host default
	mc := hostModCache()
	if mc == "" {
		t.Skip("no host GOMODCACHE")
	}
	if _, err := os.Stat(mc); err != nil {
		t.Skip("host GOMODCACHE does not exist:", err)
	}
	args, err := Args(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !argsContainPair(args, "--ro-bind", filepath.Clean(mc)) {
		t.Error("host module cache is not mounted read-only at the same path")
	}
	v, ok := argAfter(args, "GOMODCACHE")
	if !ok || v != filepath.Clean(mc) {
		t.Errorf("GOMODCACHE = %q, want %q", v, mc)
	}
}

func TestArgsChdirAndCommand(t *testing.T) {
	spec := testSpec(t)
	spec.Command = []string{"/usr/local/bin/shelley", "serve", "-db", "/x.db"}
	args, err := Args(spec)
	if err != nil {
		t.Fatal(err)
	}
	if !argsContainPair(args, "--chdir", filepath.Clean(spec.Workspace)) {
		t.Error("args do not chdir into the workspace")
	}
	// The command must follow a literal "--" separator with no shell
	// interpolation.
	for i, a := range args {
		if a == "--" {
			got := strings.Join(args[i+1:], " ")
			if got != "/usr/local/bin/shelley serve -db /x.db" {
				t.Errorf("command tail = %q", got)
			}
			return
		}
	}
	t.Fatal("args missing -- separator before command")
}

func TestSelfCheck(t *testing.T) {
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bwrap not installed")
	}
	if err := SelfCheck(testSpec(t)); err != nil {
		t.Errorf("SelfCheck on a valid spec: %v", err)
	}

	spec := testSpec(t)
	spec.Workspace = filepath.Join(t.TempDir(), "missing")
	spec.GoCache = filepath.Join(t.TempDir(), "missing-cache")
	err := SelfCheck(spec)
	if err == nil {
		t.Fatal("SelfCheck accepted missing mounts")
	}
	for _, want := range []string{"workspace mount", "go build cache"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("SelfCheck error %q does not name %q", err, want)
		}
	}
}

func TestWaitSocket(t *testing.T) {
	// No socket: must time out with an error, not succeed.
	missing := filepath.Join(t.TempDir(), "none.sock")
	if err := WaitSocket(context.Background(), missing, 300*time.Millisecond); err == nil {
		t.Error("WaitSocket succeeded for a nonexistent socket")
	}

	// A plain file is not a socket: connection must fail.
	plain := filepath.Join(t.TempDir(), "plain.sock")
	if err := os.WriteFile(plain, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WaitSocket(context.Background(), plain, 300*time.Millisecond); err == nil {
		t.Error("WaitSocket succeeded for a plain file")
	}

	// A listening Unix socket must become ready.
	ln, err := net.Listen("unix", filepath.Join(t.TempDir(), "live.sock"))
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()
	if err := WaitSocket(context.Background(), ln.Addr().String(), 2*time.Second); err != nil {
		t.Errorf("WaitSocket on a live socket: %v", err)
	}
}

// --- integration tests -----------------------------------------------------
//
// These run real bubblewrap sandboxes. They are skipped under -short or when
// bwrap is unavailable; no opt-in environment variable is required.

func requireBwrap(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping sandbox integration test in -short mode")
	}
	if _, err := exec.LookPath("bwrap"); err != nil {
		t.Skip("bwrap not available:", err)
	}
}

// runSandbox runs one sandbox command to completion and returns its combined
// output.
func runSandbox(t *testing.T, spec Spec) (string, error) {
	t.Helper()
	h, err := Start(context.Background(), spec)
	if err != nil {
		return "", err
	}
	err = h.Wait()
	return h.OutputString(), err
}

func TestIntegrationFilesystemIsolation(t *testing.T) {
	requireBwrap(t)
	spec := testSpec(t)
	// Each line ends with a named marker so output interleaving cannot
	// shift the association between check and result.
	type check struct {
		name     string
		cmd      string
		wantFail bool // isolation probes must FAIL
	}
	checks := []check{
		{"home", "echo HOME=$HOME", false},
		{"ws-write", "echo marker > ws-marker.txt", false},
		{"tmp-write", "echo tmp-marker > /tmp/private.txt", false},
		{"no-users", "stat /users", true},
		{"no-real-home", "ls /home/exedev", true},
		{"no-shadow", "stat /etc/shadow", true},
		{"exedev-config", "stat /exe.dev/shelley.json", false},
		{"no-init-cmdline", "test -s /proc/1/cmdline", true},
		{"git", "git --version", false},
		{"go", "go version", false},
		{"gofmt", "gofmt -h 2>&1 | head -1", false},
		{"make", "make -v | head -1", false},
		{"rg", "rg --version | head -1", false},
		{"bash", "bash --version | head -1", false},
	}
	var parts []string
	for _, c := range checks {
		parts = append(parts, c.cmd+fmt.Sprintf(" ; echo check-%s=$?", c.name))
	}
	parts = append(parts, "echo PROC-COUNT=$(ls /proc | grep -c '^[0-9]')")
	spec.Command = []string{"/bin/bash", "-c", strings.Join(parts, " ; ")}

	out, err := runSandbox(t, spec)
	t.Logf("sandbox output:\n%s", out)
	if err != nil {
		t.Fatalf("sandbox run: %v", err)
	}

	// HOME is the synthetic one and the workspace was writable.
	if !strings.Contains(out, "HOME=/home/shelley") {
		t.Error("HOME is not /home/shelley inside the sandbox")
	}
	if data, err := os.ReadFile(filepath.Join(spec.Workspace, "ws-marker.txt")); err != nil || strings.TrimSpace(string(data)) != "marker" {
		t.Errorf("workspace marker not written through the bind: %v", err)
	}
	if data, err := os.ReadFile(filepath.Join(spec.Tmp, "private.txt")); err != nil || strings.TrimSpace(string(data)) != "tmp-marker" {
		t.Errorf("private /tmp is not host-backed by Spec.Tmp: %v", err)
	}

	results := checkResults(out)
	for _, c := range checks {
		rc, ok := results[c.name]
		if !ok {
			t.Errorf("no result marker for check %q", c.name)
			continue
		}
		if c.wantFail && rc == 0 {
			t.Errorf("check %s succeeded but must fail (isolation breach)", c.name)
		}
		if !c.wantFail && rc != 0 {
			t.Errorf("check %s failed with rc=%d (wanted success)", c.name, rc)
		}
	}

	// /proc must be sandbox-scoped: PID 1 is bwrap's init, and only a
	// handful of processes exist.
	for _, line := range strings.Split(out, "\n") {
		if n, ok := strings.CutPrefix(line, "PROC-COUNT="); ok {
			count, err := strconv.Atoi(strings.TrimSpace(n))
			if err != nil || count > 10 {
				t.Errorf("/proc shows %s numeric entries; host processes leak", n)
			}
		}
	}
}

// checkResults extracts the check-<name>=<rc> markers printed after every
// command in the integration script.
func checkResults(out string) map[string]int {
	results := map[string]int{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if v, ok := strings.CutPrefix(line, "check-"); ok {
			if name, rcStr, ok := strings.Cut(v, "="); ok {
				if n, err := strconv.Atoi(rcStr); err == nil {
					results[name] = n
				}
			}
		}
	}
	return results
}

func TestIntegrationGoBuild(t *testing.T) {
	requireBwrap(t)
	spec := testSpec(t)

	// A tiny throwaway Go module with no external dependencies, created on
	// the host, compiled inside the sandbox. The read-only host module
	// cache stays mounted to prove the toolchain can resolve it.
	mod := filepath.Join(spec.Workspace, "tiny")
	if err := os.MkdirAll(mod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mod, "go.mod"), []byte("module tinytest\n\ngo 1.26\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	main := "package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"tiny-ok\") }\n"
	if err := os.WriteFile(filepath.Join(mod, "main.go"), []byte(main), 0o644); err != nil {
		t.Fatal(err)
	}

	spec.Command = []string{"/bin/bash", "-c",
		"cd tiny && go build ./... && go run . && test -z \"$(gofmt -l .)\" && echo GO-BUILD-OK"}
	out, err := runSandbox(t, spec)
	t.Logf("sandbox output:\n%s", out)
	if err != nil {
		t.Fatalf("go build inside sandbox: %v", err)
	}
	if !strings.Contains(out, "tiny-ok") || !strings.Contains(out, "GO-BUILD-OK") {
		t.Error("tiny module did not build and run inside the sandbox")
	}
}

func TestIntegrationStopKillsGroup(t *testing.T) {
	requireBwrap(t)
	spec := testSpec(t)
	marker := "sbstop" + strconv.FormatInt(time.Now().UnixNano(), 36)
	// Two detached sleepers with a unique marker in their command line.
	// Ignoring TERM proves the forced-kill phase works, not just SIGTERM.
	spec.Command = []string{"/bin/bash", "-c",
		"trap '' TERM; sleep 300 & sleep 300 & exec sleep 300; " + marker}
	spec.Name = "stop-" + marker

	h, err := Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("launched", h)
	time.Sleep(500 * time.Millisecond)

	if err := h.Stop(2 * time.Second); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	_ = h.Wait()

	// No process with the marker may survive; give the cgroup a moment to
	// drain before checking.
	time.Sleep(300 * time.Millisecond)
	out, _ := exec.Command("pgrep", "-f", marker).Output()
	if strings.TrimSpace(string(out)) != "" {
		t.Errorf("orphan sandbox processes survived Stop: %s", out)
	}
	if h.Unit != "" {
		out, _ := exec.Command("systemctl", "--user", "list-units", h.Unit, "--no-legend").Output()
		if strings.TrimSpace(string(out)) != "" {
			t.Errorf("systemd scope %s survived Stop: %s", h.Unit, out)
		}
	}
}

func TestIntegrationDieWithParent(t *testing.T) {
	requireBwrap(t)
	if LaunchMode() != "pgid" {
		// In systemd mode the sandbox's direct parent is the user manager,
		// so --die-with-parent does not bind to our process; the cgroup
		// lifecycle is covered by TestIntegrationStopKillsGroup instead.
		t.Skip("--die-with-parent is exercised only in pgid fallback mode")
	}
	spec := testSpec(t)
	spec.Command = []string{"/bin/bash", "-c", "sleep 300"}
	h, err := Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	// SIGKILL the direct parent (bwrap). --die-with-parent must take the
	// whole sandbox down with it.
	if err := h.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = h.Wait()
	time.Sleep(300 * time.Millisecond)
	out, _ := exec.Command("pgrep", "-f", "sleep 300").Output()
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		// Only flag survivors whose parent chain is gone: a coarse check
		// is enough because the test created the only "sleep 300" around.
		t.Errorf("sandbox child %s survived the parent's SIGKILL", line)
	}
}
