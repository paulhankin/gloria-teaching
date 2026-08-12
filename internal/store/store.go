// Package store keeps worksheet requests and site settings in SQLite.
package store

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Kind distinguishes the two request types.
type Kind string

const (
	// KindNew is a request for a brand new worksheet.
	KindNew Kind = "new"
	// KindChange is a change request for an existing worksheet.
	KindChange Kind = "change"
)

// Status is where a request stands in the pipeline.
type Status string

const (
	// StatusQueued waits for its lane to become free.
	StatusQueued Status = "queued"
	// StatusWorking means the agent is working on it right now.
	StatusWorking Status = "working"
	// StatusReview means the work is done and waits for approve/reject/refine.
	StatusReview Status = "review"
	// StatusFailed means the run (or the merge) did not work out.
	StatusFailed Status = "failed"
	// StatusDone means the change was committed and pushed.
	StatusDone Status = "done"
	// StatusRejected means the change was thrown away.
	StatusRejected Status = "rejected"
)

// Open reports whether the request still occupies its lane.
func (s Status) Open() bool {
	return s == StatusQueued || s == StatusWorking || s == StatusReview || s == StatusFailed
}

// Request is a single wish submitted through the website; it doubles as the
// work item the agent processes.
type Request struct {
	ID        int64
	Kind      Kind
	Worksheet string // worksheet path ("math/venn_diagrams"), empty for KindNew
	Author    string // optional
	Body      string
	CreatedAt time.Time

	Status     Status
	ConvID     string // shelley conversation driving this item
	Branch     string // git branch with the work
	Worktree   string // git worktree directory (absolute)
	Note       string // last status message (agent summary or error)
	HasPreview bool   // preview build available under data/preview/<id>
	UpdatedAt  time.Time
}

// Lane is the queue this request belongs to. Work in different lanes runs in
// parallel, work in the same lane strictly sequentially.
func (r Request) Lane() string {
	if r.Kind == KindChange {
		return "sheet:" + r.Worksheet
	}
	return fmt.Sprintf("new:%d", r.ID)
}

// DB is the request database.
type DB struct{ sql *sql.DB }

// Open opens (and migrates) the database at path.
func Open(path string) (*DB, error) {
	d, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	if err := migrate(d); err != nil {
		d.Close()
		return nil, err
	}
	return &DB{sql: d}, nil
}

// Close closes the database.
func (db *DB) Close() error { return db.sql.Close() }

func migrate(d *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS requests (
  id         INTEGER PRIMARY KEY AUTOINCREMENT,
  kind       TEXT NOT NULL,
  worksheet  TEXT NOT NULL DEFAULT '',
  author     TEXT NOT NULL DEFAULT '',
  body       TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  status     TEXT NOT NULL DEFAULT 'queued',
  conv_id    TEXT NOT NULL DEFAULT '',
  branch     TEXT NOT NULL DEFAULT '',
  worktree   TEXT NOT NULL DEFAULT '',
  note       TEXT NOT NULL DEFAULT '',
  preview    INTEGER NOT NULL DEFAULT 0,
  updated_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS requests_worksheet ON requests(worksheet);
CREATE TABLE IF NOT EXISTS settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);`
	if _, err := d.Exec(schema); err != nil {
		return err
	}
	// Databases created before the pipeline existed lack these columns.
	for _, col := range []string{
		"status TEXT NOT NULL DEFAULT 'queued'",
		"conv_id TEXT NOT NULL DEFAULT ''",
		"branch TEXT NOT NULL DEFAULT ''",
		"worktree TEXT NOT NULL DEFAULT ''",
		"note TEXT NOT NULL DEFAULT ''",
		"preview INTEGER NOT NULL DEFAULT 0",
		"updated_at INTEGER NOT NULL DEFAULT 0",
	} {
		name := strings.Fields(col)[0]
		if _, err := d.Exec("ALTER TABLE requests ADD COLUMN " + col); err != nil &&
			!strings.Contains(err.Error(), "duplicate column") {
			return fmt.Errorf("store: adding column %s: %w", name, err)
		}
	}
	return nil
}

// Add stores a new request and returns its ID.
func (db *DB) Add(r Request) (int64, error) {
	if r.Kind != KindNew && r.Kind != KindChange {
		return 0, fmt.Errorf("store: unknown kind %q", r.Kind)
	}
	now := time.Now().Unix()
	res, err := db.sql.Exec(
		`INSERT INTO requests (kind, worksheet, author, body, created_at, status, updated_at)
		 VALUES (?,?,?,?,?,?,?)`,
		string(r.Kind), r.Worksheet, r.Author, r.Body, now, string(StatusQueued), now)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Get returns one request.
func (db *DB) Get(id int64) (Request, error) {
	rows, err := db.sql.Query(selectRequests+` WHERE id = ?`, id)
	if err != nil {
		return Request{}, err
	}
	defer rows.Close()
	out, err := scanRequests(rows)
	if err != nil {
		return Request{}, err
	}
	if len(out) == 0 {
		return Request{}, fmt.Errorf("store: no request %d", id)
	}
	return out[0], nil
}

// SetStatus updates status and note of a request.
func (db *DB) SetStatus(id int64, s Status, note string) error {
	_, err := db.sql.Exec(
		`UPDATE requests SET status = ?, note = ?, updated_at = ? WHERE id = ?`,
		string(s), note, time.Now().Unix(), id)
	return err
}

// SetRun records the agent bookkeeping for a request.
func (db *DB) SetRun(id int64, convID, branch, worktree string) error {
	_, err := db.sql.Exec(
		`UPDATE requests SET conv_id = ?, branch = ?, worktree = ?, updated_at = ? WHERE id = ?`,
		convID, branch, worktree, time.Now().Unix(), id)
	return err
}

// SetPreview records whether a preview build exists.
func (db *DB) SetPreview(id int64, ok bool) error {
	v := 0
	if ok {
		v = 1
	}
	_, err := db.sql.Exec(`UPDATE requests SET preview = ?, updated_at = ? WHERE id = ?`,
		v, time.Now().Unix(), id)
	return err
}

// AppendBody adds a refinement to the request body.
func (db *DB) AppendBody(id int64, extra string) error {
	_, err := db.sql.Exec(
		`UPDATE requests SET body = body || ?, updated_at = ? WHERE id = ?`,
		"\n\n"+extra, time.Now().Unix(), id)
	return err
}

// Delete removes a request.
func (db *DB) Delete(id int64) error {
	_, err := db.sql.Exec(`DELETE FROM requests WHERE id = ?`, id)
	return err
}

const selectRequests = `SELECT id, kind, worksheet, author, body, created_at,
 status, conv_id, branch, worktree, note, preview, updated_at FROM requests`

func scanRequests(rows *sql.Rows) ([]Request, error) {
	var out []Request
	for rows.Next() {
		var r Request
		var created, updated int64
		var kind, status string
		var preview int
		if err := rows.Scan(&r.ID, &kind, &r.Worksheet, &r.Author, &r.Body, &created,
			&status, &r.ConvID, &r.Branch, &r.Worktree, &r.Note, &preview, &updated); err != nil {
			return nil, err
		}
		r.Kind = Kind(kind)
		r.Status = Status(status)
		r.HasPreview = preview != 0
		r.CreatedAt = time.Unix(created, 0)
		r.UpdatedAt = time.Unix(updated, 0)
		out = append(out, r)
	}
	return out, rows.Err()
}

// All returns every request, newest first.
func (db *DB) All() ([]Request, error) {
	rows, err := db.sql.Query(selectRequests + ` ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRequests(rows)
}

// Active returns every request that is not finished, oldest first.
func (db *DB) Active() ([]Request, error) {
	rows, err := db.sql.Query(selectRequests+
		` WHERE status NOT IN (?,?) ORDER BY id ASC`, string(StatusDone), string(StatusRejected))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRequests(rows)
}

// Setting returns a stored setting, or def when it is not set.
func (db *DB) Setting(key, def string) string {
	var v string
	err := db.sql.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if err != nil {
		return def
	}
	return v
}

// SetSetting stores a setting.
func (db *DB) SetSetting(key, value string) error {
	_, err := db.sql.Exec(
		`INSERT INTO settings (key, value) VALUES (?,?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// AdminMode reports whether the front page shows the submitted requests.
func (db *DB) AdminMode() bool { return db.Setting("admin_mode", "off") == "on" }

// SetAdminMode enables or disables admin mode.
func (db *DB) SetAdminMode(on bool) error {
	v := "off"
	if on {
		v = "on"
	}
	return db.SetSetting("admin_mode", v)
}
