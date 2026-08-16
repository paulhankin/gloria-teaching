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

// User is an account that can authenticate with an email address and password.
type User struct {
	ID           int64
	Email        string
	PasswordHash []byte
	Verified     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// AuthToken is a one-time email verification or password-reset token.
type AuthToken struct {
	UserID    int64
	Purpose   string
	ExpiresAt time.Time
}

// DB is the request and account database.
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
);
CREATE TABLE IF NOT EXISTS users (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  email         TEXT NOT NULL COLLATE NOCASE UNIQUE,
  password_hash BLOB NOT NULL,
  verified_at   INTEGER NOT NULL DEFAULT 0,
  created_at    INTEGER NOT NULL,
  updated_at    INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS auth_tokens (
  token_hash BLOB PRIMARY KEY,
  user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  purpose    TEXT NOT NULL,
  expires_at INTEGER NOT NULL,
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS auth_tokens_user ON auth_tokens(user_id, purpose);`
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

// CreateUser creates an unverified account.
func (db *DB) CreateUser(email string, passwordHash []byte) (User, error) {
	now := time.Now().Unix()
	res, err := db.sql.Exec(
		`INSERT INTO users (email, password_hash, created_at, updated_at) VALUES (?,?,?,?)`,
		email, passwordHash, now, now)
	if err != nil {
		return User{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, err
	}
	return db.UserByID(id)
}

// UserByID returns an account by ID.
func (db *DB) UserByID(id int64) (User, error) {
	return db.scanUser(`SELECT id, email, password_hash, verified_at, created_at, updated_at FROM users WHERE id = ?`, id)
}

// UserByEmail returns an account by its case-insensitive email address.
func (db *DB) UserByEmail(email string) (User, error) {
	return db.scanUser(`SELECT id, email, password_hash, verified_at, created_at, updated_at FROM users WHERE email = ?`, email)
}

func (db *DB) scanUser(query string, arg any) (User, error) {
	var u User
	var verified, created, updated int64
	err := db.sql.QueryRow(query, arg).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &verified, &created, &updated)
	if err != nil {
		return User{}, err
	}
	u.Verified = verified != 0
	u.CreatedAt = time.Unix(created, 0)
	u.UpdatedAt = time.Unix(updated, 0)
	return u, nil
}

// MarkUserVerified marks an account's email address as verified.
func (db *DB) MarkUserVerified(id int64) error {
	now := time.Now().Unix()
	_, err := db.sql.Exec(
		`UPDATE users SET verified_at = CASE WHEN verified_at = 0 THEN ? ELSE verified_at END, updated_at = ? WHERE id = ?`,
		now, now, id)
	return err
}

// SetUserPassword changes an account password.
func (db *DB) SetUserPassword(id int64, passwordHash []byte) error {
	_, err := db.sql.Exec(`UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`,
		passwordHash, time.Now().Unix(), id)
	return err
}

// PutAuthToken replaces a user's token for one purpose.
func (db *DB) PutAuthToken(userID int64, purpose string, tokenHash []byte, expiresAt time.Time) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM auth_tokens WHERE user_id = ? AND purpose = ?`, userID, purpose); err != nil {
		return err
	}
	if _, err := tx.Exec(
		`INSERT INTO auth_tokens (token_hash, user_id, purpose, expires_at, created_at) VALUES (?,?,?,?,?)`,
		tokenHash, userID, purpose, expiresAt.Unix(), time.Now().Unix()); err != nil {
		return err
	}
	return tx.Commit()
}

// ConsumeAuthToken removes and returns a valid one-time token.
func (db *DB) ConsumeAuthToken(tokenHash []byte, purpose string) (AuthToken, error) {
	tx, err := db.sql.Begin()
	if err != nil {
		return AuthToken{}, err
	}
	defer tx.Rollback()
	var t AuthToken
	var expires int64
	err = tx.QueryRow(
		`SELECT user_id, purpose, expires_at FROM auth_tokens WHERE token_hash = ? AND purpose = ?`,
		tokenHash, purpose).Scan(&t.UserID, &t.Purpose, &expires)
	if err != nil {
		return AuthToken{}, err
	}
	if time.Now().Unix() > expires {
		_, _ = tx.Exec(`DELETE FROM auth_tokens WHERE token_hash = ?`, tokenHash)
		if err := tx.Commit(); err != nil {
			return AuthToken{}, err
		}
		return AuthToken{}, sql.ErrNoRows
	}
	if _, err := tx.Exec(`DELETE FROM auth_tokens WHERE token_hash = ?`, tokenHash); err != nil {
		return AuthToken{}, err
	}
	if err := tx.Commit(); err != nil {
		return AuthToken{}, err
	}
	t.ExpiresAt = time.Unix(expires, 0)
	return t, nil
}

// DeleteExpiredAuthTokens removes stale verification and reset links.
func (db *DB) DeleteExpiredAuthTokens() error {
	_, err := db.sql.Exec(`DELETE FROM auth_tokens WHERE expires_at < ?`, time.Now().Unix())
	return err
}
