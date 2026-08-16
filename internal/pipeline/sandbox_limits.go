package pipeline

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// Workspace size limiting for sandbox mode. The budget covers the WHOLE
// request directory (data/sandboxes/req-<id>): the workspace clones, the
// synthetic home (including the Go build cache the agent's builds fill), the
// isolated Shelley state, and tmp. The plan defines a "maximum
// request-directory size"; limiting only workspace/ would leave the Go
// cache — the easiest place to write gigabytes — unbounded.

// dirSize returns the total size in bytes of everything below root. It uses
// filepath.WalkDir, which never follows symlinks; a symlink contributes its
// own directory-entry size (a few bytes) instead of its target's, so
// symlinks pointing outside the sandbox cannot hide or inflate usage. A
// missing root reports 0 bytes and the stat error.
func dirSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}

// enforceWorkspaceLimit fails the request when the sandbox directory exceeds
// maxBytes. It returns nil when the limit is disabled (maxBytes <= 0, e.g. a
// manually constructed Pipeline without normalized limits) or when the
// sandbox paths are not valid (paths.Root == "" in worktree mode).
func (p *Pipeline) enforceWorkspaceLimit(paths sandboxPaths, maxBytes int64) error {
	if paths.Root == "" || maxBytes <= 0 {
		return nil
	}
	size, err := dirSize(paths.Root)
	if err != nil {
		return fmt.Errorf("measure the sandbox size of request #%d: %w", sandboxRequestID(paths), err)
	}
	if size > maxBytes {
		return fmt.Errorf("request #%d sandbox directory exceeded its %d byte limit (%d bytes used); "+
			"the isolated server was stopped and the request failed", sandboxRequestID(paths), maxBytes, size)
	}
	return nil
}
