// Package sandbox runs one process (typically the isolated "shelley serve"
// binary of a worksheet request) inside a bubblewrap sandbox with a strict
// filesystem allowlist.
//
// The package owns all bubblewrap mechanics: argument construction,
// environment and mount allowlists, startup self-checks, process/cgroup
// lifecycle, and socket readiness detection. The pipeline state machine only
// ever sees Spec, Handle, Start, Stop, SelfCheck, and WaitSocket.
//
// Stage 1 deliberately keeps the host network namespace: Shelley reaches the
// exe.dev LLM integration through the ambient network environment. The
// sandbox is a filesystem and process boundary, not a network boundary.
package sandbox

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// Limits bounds the resource consumption of one sandbox. It is applied as
// systemd scope properties when the user manager is available. The pipeline
// converts its own SandboxLimits into this type so the sandbox package stays
// pipeline-free. An empty value means "no limit" for the field.
type Limits struct {
	MemoryMax string // cgroup memory.max, e.g. "4G"
	TasksMax  int    // cgroup pids.max
	CPUQuota  string // cgroup cpu.max quota, e.g. "200%"
}

// Spec describes one sandbox launch.
type Spec struct {
	Name      string   // request name, e.g. "req-42" (used in unit name/diagnostics)
	Workspace string   // absolute path; mounted read-write at the same path
	Home      string   // synthetic HOME; mounted read-write at /home/shelley
	State     string   // request-local shelley state dir; read-write at the same path
	Tmp       string   // private host-backed tmp dir; mounted read-write at /tmp
	GoCache   string   // mounted read-write (GOCACHE points at its sandbox view)
	Command   []string // argv executed inside the sandbox
	Env       []string // EXTRA allowed env (KEY=VALUE), appended to the fixed minimal set
	Limits    Limits   // resource limits applied via systemd scope properties

	// ModCache is a read-only Go module cache bind. When empty it defaults to
	// hostModCache() (go env GOMODCACHE). Set it to "-" to disable the mount.
	// The sandbox GOMODCACHE points at the sandbox-visible path.
	ModCache string
	// ShelleyPath overrides the shelley executable resolved by exec.LookPath;
	// it is mounted read-only if it is not already covered by /usr.
	ShelleyPath string
}

// unitPrefix is prepended to every transient systemd scope name so stale
// units are easy to find and kill.
const unitPrefix = "learningmaterial-sandbox-"

// homeMount is where Spec.Home is mounted inside the sandbox.
const homeMount = "/home/shelley"

// fixedEnv returns the minimal fixed environment inside the sandbox. The
// PATH only lists directories that actually exist inside the sandbox.
func fixedEnv(spec Spec) []string {
	return []string{
		"HOME=" + homeMount,
		"PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin",
		"GOCACHE=" + goCacheSandboxPath(spec),
		"GOPATH=" + homeMount + "/go",
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
		"TMPDIR=/tmp",
		"GOMODCACHE=" + modCacheSandboxPath(spec),
	}
}

// sensitiveEnvSubstrings are matched case-insensitively against names of
// Spec.Env extras. Any match rejects the whole Spec: the sandbox must never
// receive credentials, proxy settings, or hooks that let a child process
// silently authenticate somewhere.
var sensitiveEnvSubstrings = []string{
	"KEY", "TOKEN", "SECRET", "PASSWORD", "CREDENTIAL",
	"SSH_", "SHELLEY_URL", "PROXY", "GIT_ASKPASS",
}

// checkExtraEnv validates one KEY=VALUE extra environment entry.
func checkExtraEnv(entry string) error {
	name, _, ok := strings.Cut(entry, "=")
	if !ok || name == "" {
		return fmt.Errorf("environment entry %q is not KEY=VALUE", entry)
	}
	upper := strings.ToUpper(name)
	for _, bad := range sensitiveEnvSubstrings {
		if strings.Contains(upper, bad) {
			return fmt.Errorf("environment variable %q rejected: name matches sensitive pattern %q", name, bad)
		}
	}
	switch upper {
	case "HOME", "PATH", "GOCACHE", "GOPATH", "GOMODCACHE", "GOFLAGS", "LANG", "LC_ALL", "TMPDIR":
		return fmt.Errorf("environment variable %q is fixed by the sandbox policy", name)
	}
	return nil
}

// goCacheSandboxPath returns the path of the writable Go build cache as seen
// inside the sandbox. A cache below the synthetic home is covered by the home
// bind; anything else is bind-mounted at its host path.
func goCacheSandboxPath(spec Spec) string {
	if spec.GoCache == "" {
		return homeMount + "/.cache/go-build"
	}
	if belowHome(spec) {
		return homeMount + strings.TrimPrefix(filepath.Clean(spec.GoCache), filepath.Clean(spec.Home))
	}
	return filepath.Clean(spec.GoCache)
}

// belowHome reports whether the Go build cache lives below the synthetic
// home (and is therefore already writable through the home bind).
func belowHome(spec Spec) bool {
	home := filepath.Clean(spec.Home)
	cache := filepath.Clean(spec.GoCache)
	return cache == home || strings.HasPrefix(cache, home+string(filepath.Separator))
}

// modCacheMount is where the read-only Go module cache is mounted inside the
// sandbox. A fixed location below the synthetic home keeps the host path
// (typically /home/<user>/go/pkg/mod) out of the sandbox mount table:
// mounting it at its host path would make bubblewrap auto-create the parent
// /home/<user>, leaking the host home directory's existence into the
// sandbox.
const modCacheMount = homeMount + "/.gomodcache"

// modCacheSandboxPath returns the sandbox-visible path of the read-only Go
// module cache. When the cache is disabled ("-") or the directory does not
// exist (the bind is skipped), it falls back to the sandbox-local default
// below GOPATH.
func modCacheSandboxPath(spec Spec) string {
	if spec.ModCache == "-" {
		return homeMount + "/go/pkg/mod"
	}
	mc := spec.ModCache
	if mc == "" {
		mc = hostModCache()
	}
	if mc == "" {
		return homeMount + "/go/pkg/mod"
	}
	if _, err := os.Stat(mc); err != nil {
		return homeMount + "/go/pkg/mod"
	}
	return modCacheMount
}

// hostModCache returns the host Go module cache. exe.dev VMs pin GOFLAGS
// (including -mod=mod) system-wide via /etc/environment, so the result of
// "go env GOMODCACHE" is authoritative for builds inside the sandbox too.
func hostModCache() string {
	if out, err := exec.Command("go", "env", "GOMODCACHE").Output(); err == nil {
		if s := strings.TrimSpace(string(out)); s != "" {
			return s
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, "go", "pkg", "mod")
	}
	return ""
}

// hostEnv returns one variable from the current process environment. It is a
// function variable so tests can inject a fake environment.
func hostEnv(key string) string { return os.Getenv(key) }

// validateSpec checks that the mandatory pieces of a Spec are present and
// absolute, and that no extra environment entry is sensitive.
func validateSpec(spec *Spec) error {
	if spec.Name == "" {
		return fmt.Errorf("sandbox name is empty")
	}
	for _, r := range spec.Name {
		ok := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' ||
			r == '-' || r == '_' || r == '.'
		if !ok {
			return fmt.Errorf("sandbox name %q contains invalid character %q", spec.Name, r)
		}
	}
	for name, path := range map[string]string{
		"workspace": spec.Workspace,
		"home":      spec.Home,
		"state":     spec.State,
		"tmp":       spec.Tmp,
	} {
		if path == "" {
			return fmt.Errorf("sandbox %s path is empty", name)
		}
		if !filepath.IsAbs(path) {
			return fmt.Errorf("sandbox %s path %q is not absolute", name, path)
		}
	}
	if spec.GoCache != "" && !filepath.IsAbs(spec.GoCache) {
		return fmt.Errorf("sandbox go cache path %q is not absolute", spec.GoCache)
	}
	if len(spec.Command) == 0 {
		return fmt.Errorf("sandbox command is empty")
	}
	for _, entry := range spec.Env {
		if err := checkExtraEnv(entry); err != nil {
			return err
		}
	}
	if spec.ModCache == "" {
		spec.ModCache = hostModCache()
	}
	return nil
}

// Args builds the bubblewrap argument slice for spec. The returned slice
// contains neither the bwrap binary nor any shell: callers exec bwrap with
// these arguments directly. Bind sources that do not exist on the host are
// skipped by design (SelfCheck verifies the mandatory ones).
func Args(spec Spec) ([]string, error) {
	if err := validateSpec(&spec); err != nil {
		return nil, err
	}

	args := []string{
		// Namespace and process isolation. The network namespace is
		// deliberately shared in Stage 1 (exe.dev LLM integration).
		"--unshare-user",
		"--unshare-pid",
		"--unshare-ipc",
		"--unshare-uts",
		"--unshare-cgroup-try",
		"--disable-userns",
		"--new-session",
		"--die-with-parent",
		"--cap-drop", "ALL",

		// Fresh /proc and minimal /dev. No /sys, /run, host /tmp, or any
		// host socket is ever added below.
		"--proc", "/proc",
		"--dev", "/dev",

		// Inside the PID namespace bwrap's init IS pid 1, so /proc/1/*
		// exists; mask its interesting entries so no sandbox process can
		// read the launcher's command line or environ.
		"--ro-bind", "/dev/null", "/proc/1/cmdline",
		"--ro-bind", "/dev/null", "/proc/1/environ",

		// A tmpfs /home ensures no bind destination or chdir path below
		// /home ever auto-creates a directory that would expose the host's
		// /home listing (bubblewrap auto-creates parent directories of
		// bind destinations, so without this /home/exedev would be
		// visible). The synthetic home bind follows below.
		"--tmpfs", "/home",
	}

	// Environment: start clean, then allow only the fixed minimal set plus
	// validated extras.
	args = append(args, "--clearenv")
	for _, kv := range fixedEnv(spec) {
		name, value, _ := strings.Cut(kv, "=")
		args = append(args, "--setenv", name, value)
	}
	for _, kv := range spec.Env {
		name, value, _ := strings.Cut(kv, "=")
		args = append(args, "--setenv", name, value)
	}

	// Read-only base system. /bin, /lib, /lib64, and /sbin are symlinks
	// into /usr on this host, so recreating them as symlinks is correct.
	args = append(args,
		"--ro-bind", "/usr", "/usr",
		"--symlink", "usr/bin", "/bin",
		"--symlink", "usr/lib", "/lib",
		"--symlink", "usr/lib64", "/lib64",
		"--symlink", "usr/sbin", "/sbin",
	)

	// Selected /etc files: TLS certificates plus DNS/NSS glue. Never all of
	// /etc. Symlinked sources are resolved so the bind does not fail inside
	// the restricted namespace.
	var notes []string
	for _, f := range []string{
		"/etc/resolv.conf",
		"/etc/hosts",
		"/etc/nsswitch.conf",
		"/etc/ssl/certs",
		"/etc/ca-certificates",
		"/etc/passwd",
		"/etc/group",
		"/etc/localtime",
	} {
		args = roBindIfExists(args, f, f, &notes)
	}

	// exe.dev Shelley configuration: the isolated server needs it.
	args = roBindIfExists(args, "/exe.dev/shelley.json", "/exe.dev/shelley.json", &notes)

	// The shelley executable, unless /usr already covers it.
	shelley := spec.ShelleyPath
	if shelley == "" {
		shelley, _ = exec.LookPath("shelley")
	}
	if shelley != "" {
		if resolved, err := filepath.EvalSymlinks(shelley); err == nil {
			shelley = resolved
		}
		if !strings.HasPrefix(shelley, "/usr/") {
			args = roBindIfExists(args, shelley, shelley, &notes)
		}
	}

	// Go toolchain and module cache (read-only; the writable build cache is
	// separate). GOFLAGS=-mod=mod keeps "go build" usable with a read-only
	// module cache on Go >= 1.16. The module cache is mounted at the fixed
	// modCacheMount (see modCacheSandboxPath), never at its host path.
	args = roBindIfExists(args, "/usr/local/go", "/usr/local/go", &notes)
	if spec.ModCache != "-" && spec.ModCache != "" {
		mc := filepath.Clean(spec.ModCache)
		args = roBindIfExists(args, mc, modCacheMount, &notes)
	}

	// Writable request mounts. Mount ORDER IS LOAD-BEARING:
	//
	//  1. bubblewrap auto-creates parent directories of bind destinations
	//     in the CURRENT root, so every destination below /tmp must be
	//     mounted after the private /tmp bind, and /home/shelley needs its
	//     /home parent present (the --tmpfs /home above provides it).
	//  2. The home bind precedes the Go build cache bind in case the cache
	//     lives below home.
	args = bindIfExists(args, spec.Home, homeMount, &notes)
	args = bindIfExists(args, spec.Tmp, "/tmp", &notes)
	args = bindIfExists(args, spec.Workspace, filepath.Clean(spec.Workspace), &notes)
	args = bindIfExists(args, spec.State, filepath.Clean(spec.State), &notes)
	if spec.GoCache != "" && !belowHome(spec) {
		args = bindIfExists(args, spec.GoCache, filepath.Clean(spec.GoCache), &notes)
	}

	// Working directory inside the sandbox.
	args = append(args, "--chdir", filepath.Clean(spec.Workspace))

	args = append(args, "--")
	args = append(args, spec.Command...)
	return args, nil
}

// roBindIfExists appends a read-only bind of src at dst when src exists on
// the host; otherwise it records a note. Symlinks in src are resolved first.
func roBindIfExists(args []string, src, dst string, notes *[]string) []string {
	resolved, err := resolveExisting(src)
	if err != nil {
		*notes = append(*notes, fmt.Sprintf("skipped read-only mount %s: %v", src, err))
		return args
	}
	return append(args, "--ro-bind", resolved, dst)
}

// bindIfExists appends a read-write bind of src at dst when src exists.
func bindIfExists(args []string, src, dst string, notes *[]string) []string {
	resolved, err := resolveExisting(src)
	if err != nil {
		*notes = append(*notes, fmt.Sprintf("skipped writable mount %s: %v", src, err))
		return args
	}
	return append(args, "--bind", resolved, dst)
}

// resolveExisting returns the canonical absolute path of src, or an error if
// src does not exist.
func resolveExisting(src string) (string, error) {
	if _, err := os.Stat(src); err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(src)
	if err != nil {
		return "", err
	}
	return resolved, nil
}

// requiredTools are the executables that must be runnable inside every
// sandbox, looked up in the sandbox PATH order.
var requiredTools = []string{"bash", "git", "go", "gofmt", "make", "rg"}

// SelfCheck verifies that the sandbox can actually start: bwrap, systemd-run,
// and every required tool must exist, and the mandatory mounts must be
// present. It returns a concise diagnostic naming everything that is
// missing.
func SelfCheck(spec Spec) error {
	var missing []string

	if _, err := exec.LookPath("bwrap"); err != nil {
		missing = append(missing, "executable bwrap")
	}
	if _, err := exec.LookPath("systemd-run"); err != nil {
		missing = append(missing, "executable systemd-run")
	}

	if err := validateSpec(&spec); err != nil {
		missing = append(missing, err.Error())
	}

	// Required tools: /usr is always mounted, /usr/local/go is mounted when
	// it exists; check the same lookup order the sandbox PATH uses.
	toolDirs := []string{"/usr/local/go/bin", "/usr/local/bin", "/usr/bin", "/bin"}
	for _, tool := range requiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			found := false
			for _, dir := range toolDirs {
				if _, err := os.Stat(filepath.Join(dir, tool)); err == nil {
					found = true
					break
				}
			}
			if !found {
				missing = append(missing, "executable "+tool)
			}
		}
	}

	// Mandatory mounts.
	for name, path := range map[string]string{
		"workspace mount": spec.Workspace,
		"home mount":      spec.Home,
		"state mount":     spec.State,
		"tmp mount":       spec.Tmp,
		"/usr":            "/usr",
		"go toolchain":    "/usr/local/go",
		"exe.dev config":  "/exe.dev/shelley.json",
		"resolv.conf":     "/etc/resolv.conf",
		"nsswitch.conf":   "/etc/nsswitch.conf",
		"TLS certs":       "/etc/ssl/certs",
	} {
		if path != "" {
			if _, err := os.Stat(path); err != nil {
				missing = append(missing, fmt.Sprintf("%s (%s)", name, path))
			}
		}
	}
	if spec.GoCache != "" {
		if _, err := os.Stat(spec.GoCache); err != nil {
			missing = append(missing, fmt.Sprintf("go build cache (%s)", spec.GoCache))
		}
	}
	if spec.ModCache != "" && spec.ModCache != "-" {
		if _, err := os.Stat(spec.ModCache); err != nil {
			missing = append(missing, fmt.Sprintf("go module cache (%s)", spec.ModCache))
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		return errors.New("sandbox self-check failed, missing: " + strings.Join(missing, ", "))
	}
	return nil
}

// bwrapPath locates the bubblewrap executable.
func bwrapPath() (string, error) {
	p, err := exec.LookPath("bwrap")
	if err != nil {
		return "", fmt.Errorf("bubblewrap (bwrap) not found in PATH: %w", err)
	}
	return p, nil
}

// strconvQuote is a tiny helper to keep systemd property values unambiguous.
func strconvQuote(s string) string { return strconv.Quote(s) }

// argAfter returns the argument following the first occurrence of flag, or
// false when flag is absent. Used by tests and diagnostics.
func argAfter(args []string, flag string) (string, bool) {
	for i, a := range args {
		if a == flag && i+1 < len(args) {
			return args[i+1], true
		}
	}
	return "", false
}

// argsContainPair reports whether args contains a followed immediately by b.
func argsContainPair(args []string, a, b string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == a && args[i+1] == b {
			return true
		}
	}
	return false
}
