package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sandboxPaths holds every host path that belongs to one request sandbox.
// All paths are absolute and below Root.
type sandboxPaths struct {
	Root              string // data/sandboxes/req-<id>
	Workspace         string // Root/workspace: standalone core repository clone
	WorkspaceUserRepo string // Root/workspace/generate/<username>: standalone worksheet clone
	Home              string // Root/home: synthetic HOME visible to Shelley
	GoCache           string // Root/home/.cache/go-build
	GoConfig          string // Root/home/.config/go
	State             string // Root/state: isolated Shelley state
	DB                string // Root/state/shelley.db
	Socket            string // Root/state/shelley.sock
	ServerLog         string // Root/state/server.log
	Metadata          string // Root/state/metadata.json
	Tmp               string // Root/tmp: private host-backed temp dir
}

// sandboxPathsFor computes the sandbox layout for one request. root must be an
// absolute path and username a plain path segment (no traversal); every
// derived path is verified to stay below root.
func sandboxPathsFor(root string, id int64, username string) (sandboxPaths, error) {
	if root == "" {
		return sandboxPaths{}, fmt.Errorf("sandbox root is not configured")
	}
	if !filepath.IsAbs(root) {
		return sandboxPaths{}, fmt.Errorf("sandbox root %q is not absolute", root)
	}
	if err := validateSandboxUsername(username); err != nil {
		return sandboxPaths{}, err
	}

	cleanRoot := filepath.Clean(root)
	sb := sandboxPaths{Root: filepath.Join(cleanRoot, fmt.Sprintf("req-%d", id))}
	sb.Workspace = filepath.Join(sb.Root, "workspace")
	sb.WorkspaceUserRepo = filepath.Join(sb.Workspace, "generate", username)
	sb.Home = filepath.Join(sb.Root, "home")
	sb.GoCache = filepath.Join(sb.Home, ".cache", "go-build")
	sb.GoConfig = filepath.Join(sb.Home, ".config", "go")
	sb.State = filepath.Join(sb.Root, "state")
	sb.DB = filepath.Join(sb.State, "shelley.db")
	sb.Socket = filepath.Join(sb.State, "shelley.sock")
	sb.ServerLog = filepath.Join(sb.State, "server.log")
	sb.Metadata = filepath.Join(sb.State, "metadata.json")
	sb.Tmp = filepath.Join(sb.Root, "tmp")

	for name, path := range map[string]string{
		"root":           sb.Root,
		"workspace":      sb.Workspace,
		"user repo":      sb.WorkspaceUserRepo,
		"home":           sb.Home,
		"go build cache": sb.GoCache,
		"go config":      sb.GoConfig,
		"state":          sb.State,
		"database":       sb.DB,
		"socket":         sb.Socket,
		"server log":     sb.ServerLog,
		"metadata":       sb.Metadata,
		"tmp":            sb.Tmp,
	} {
		if !pathBelow(cleanRoot, path) {
			return sandboxPaths{}, fmt.Errorf("sandbox %s path %q escapes sandbox root %q", name, path, cleanRoot)
		}
	}
	return sb, nil
}

// validateSandboxUsername mirrors the username validation in requestUsername:
// the username becomes a path segment, so it must not contain separators or
// traversal elements.
func validateSandboxUsername(username string) error {
	if username == "" || username == "." || username == ".." || strings.ContainsAny(username, `/\\`) {
		return fmt.Errorf("invalid sandbox username %q", username)
	}
	if filepath.Clean(username) != username {
		return fmt.Errorf("invalid sandbox username %q", username)
	}
	return nil
}

// pathBelow reports whether path is root itself or a descendant of root.
// Both must be clean absolute paths.
func pathBelow(root, path string) bool {
	if path == root {
		return true
	}
	prefix := root
	if !strings.HasSuffix(prefix, string(filepath.Separator)) {
		prefix += string(filepath.Separator)
	}
	return strings.HasPrefix(path, prefix)
}

// createSandboxDirs creates the sandbox directories that must exist before
// the clones and the Shelley server are set up.
func (p *Pipeline) createSandboxDirs(paths sandboxPaths) error {
	for _, dir := range []string{paths.Home, paths.GoCache, paths.GoConfig, paths.State, paths.Tmp} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create sandbox directory %s: %w", dir, err)
		}
	}
	return nil
}
