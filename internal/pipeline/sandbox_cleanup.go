package pipeline

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"learningmaterial/internal/store"
)

// Retention and cleanup policy for sandbox mode (plan step 6).
//
// Publication already removes a request's sandbox synchronously
// (cleanupWorkspace in publish). Everything else is handled by the janitor:
// a background goroutine (sandbox mode only, ~10-minute ticker) scanning
// SandboxRoot for req-<id> directories. A directory is removed when its
// request is in a terminal state and its metadata is older than the
// configured retention: done -> RetainCompleted (short, for debugging),
// failed/rejected -> RetainFailed (longer, to explain what happened), no
// request row at all -> RetainFailed. The age is the metadata file's mtime:
// every phase transition rewrites metadata, so it reflects the last
// activity. Before deleting, the janitor sets the cleanup phase in the
// metadata (best effort; the directory deletion is the end state anyway).
//
// A sandbox is NEVER deleted while its request is active (queued, working,
// review, or failed-but-retryable while a run goroutine or a registered
// server holds it): active requests keep their directory no matter how old.
//
// cleanStaleSandboxes runs the same scan once at startup and additionally
// kills every recorded live process (killStaleSandbox) BEFORE the pipeline
// requeues work, covering sandboxes whose request row is gone or closed —
// recover() only revisits rows the database still lists as active.

// janitor sweeps SandboxRoot every interval until the process exits. It runs
// once per interval; JanitorRun is the exported single sweep (used by the
// janitor and by tests).
func (p *Pipeline) janitor(interval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	for range tick.C {
		p.JanitorRun(time.Now())
	}
}

// cleanStaleSandboxes is the startup sweep: terminate every sandbox process
// recorded under SandboxRoot (even ones whose request row is gone or closed)
// and apply the retention policy. Errors are logged, never fatal.
func (p *Pipeline) cleanStaleSandboxes() {
	if p.db == nil {
		return
	}
	now := time.Now()
	for _, sb := range p.scanSandboxes(now) {
		if sb.meta.PID > 0 || sb.meta.Unit != "" {
			if err := killStaleSandbox(sb.meta); err != nil {
				p.Logf("startup cleanup: kill the stale sandbox of request #%d: %v", sb.id, err)
			}
		}
		// Never delete the sandbox of an active request at startup: recover()
		// is about to requeue or republish it. Terminal/orphaned sandboxes
		// are deleted immediately: their processes are dead now and the
		// retention clock already ran while the service was down.
		if sb.deletable(now, p.cfg.Limits, true) {
			p.removeScannedSandbox(sb, "startup")
		}
	}
}

// JanitorRun applies the retention policy once. now is the reference time
// (tests pass a fixed clock).
func (p *Pipeline) JanitorRun(now time.Time) {
	if p.db == nil {
		return
	}
	for _, sb := range p.scanSandboxes(now) {
		if !sb.deletable(now, p.cfg.Limits, false) {
			continue
		}
		if sb.meta.PID > 0 || sb.meta.Unit != "" {
			// A stale process of a terminal request must never survive its
			// sandbox's deletion.
			if err := killStaleSandbox(sb.meta); err != nil {
				p.Logf("janitor: kill the stale sandbox of request #%d: %v", sb.id, err)
			}
		}
		p.removeScannedSandbox(sb, "janitor")
	}
}

// scannedSandbox is one req-<id> directory found below SandboxRoot.
type scannedSandbox struct {
	id     int64
	root   string
	meta   sandboxMetadata
	hasDB  bool          // a request row exists
	status store.Status  // request status; "" when the row is gone
	age    time.Duration // now - metadata mtime (last activity)
	active bool          // a run goroutine or a registered server owns it
}

// scanSandboxes lists every well-formed sandbox below SandboxRoot with the
// data the retention policy needs. Malformed entries are logged and skipped.
func (p *Pipeline) scanSandboxes(now time.Time) []scannedSandbox {
	root := p.cfg.SandboxRoot
	if root == "" || !filepath.IsAbs(root) {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			p.Logf("sandbox scan: %v", err)
		}
		return nil
	}
	var out []scannedSandbox
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "req-") {
			continue
		}
		id, err := strconv.ParseInt(strings.TrimPrefix(entry.Name(), "req-"), 10, 64)
		if err != nil || id <= 0 {
			p.Logf("sandbox scan: ignoring malformed sandbox directory %q", entry.Name())
			continue
		}
		sb := scannedSandbox{id: id, root: filepath.Join(root, entry.Name())}

		metaPath := filepath.Join(sb.root, "state", "metadata.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			// No readable metadata: treat the directory as an orphan whose
			// age comes from the directory itself (touched at creation).
			if info, statErr := entry.Info(); statErr == nil {
				sb.age = now.Sub(info.ModTime())
			}
		} else {
			if err := json.Unmarshal(data, &sb.meta); err != nil {
				p.Logf("sandbox scan: ignoring %s with unreadable metadata: %v", sb.root, err)
				continue
			}
			if info, statErr := os.Stat(metaPath); statErr == nil {
				sb.age = now.Sub(info.ModTime())
			}
		}

		var req store.Request
		if it, err := p.db.Get(id); err == nil {
			sb.hasDB = true
			sb.status = it.Status
			req = it
		}
		p.mu.Lock()
		if _, ok := p.servers[id]; ok {
			// A registered isolated server owns this sandbox right now.
			sb.active = true
		} else if sb.hasDB && p.busy[req.Lane()] && req.Status.Open() {
			// A run goroutine holds the request's lane and the request is
			// still active. (For KindNew the lane is per-request anyway;
			// for KindChange the lane is per-worksheet and could belong to
			// a sibling request — but then THIS request is already in a
			// terminal state and the lane being busy does not protect it,
			// which is correct: retention only spares open requests.)
			sb.active = true
		}
		p.mu.Unlock()
		out = append(out, sb)
	}
	return out
}

// deletable applies the retention policy. startup relaxes the age check for
// terminal and orphaned sandboxes: their retention clock ran while the
// service was down and their processes were just killed.
func (sb scannedSandbox) deletable(now time.Time, limits SandboxLimits, startup bool) bool {
	if sb.active {
		return false // never delete a sandbox whose request is active
	}
	var retain time.Duration
	switch {
	case !sb.hasDB:
		retain = limits.RetainFailed // no request row: orphaned sandbox
	case sb.status == store.StatusDone:
		retain = limits.RetainCompleted
	case sb.status == store.StatusFailed || sb.status == store.StatusRejected:
		retain = limits.RetainFailed
	default:
		// queued/working/review: active, whatever the bookkeeping says
		return false
	}
	return startup || sb.age >= retain
}

// removeScannedSandbox best-effort records the cleanup phase in the metadata
// and deletes the sandbox directory.
func (p *Pipeline) removeScannedSandbox(sb scannedSandbox, who string) {
	meta := sb.meta
	meta.PID = 0
	meta.Unit = ""
	meta.Phase = phaseCleaning
	meta.CleanupState = "deleted by " + who
	metaPath := filepath.Join(sb.root, "state", "metadata.json")
	paths := sandboxPaths{Root: sb.root, Metadata: metaPath}
	if err := writeMetadata(paths, meta); err != nil {
		p.Logf("#%d: record the cleaning phase: %v", sb.id, err)
	}
	if err := os.RemoveAll(sb.root); err != nil {
		p.Logf("#%d: delete sandbox %s: %v", sb.id, sb.root, err)
		return
	}
	p.Logf("#%d: deleted sandbox %s (%s; status %q, age %s)",
		sb.id, sb.root, who, statusLabel(sb.hasDB, sb.status), sb.age.Round(time.Minute))
}

// statusLabel renders a request status for logs, marking gone rows.
func statusLabel(hasDB bool, s store.Status) string {
	if !hasDB {
		return "no request row"
	}
	return string(s)
}
