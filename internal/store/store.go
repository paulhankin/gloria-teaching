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
	Author    string // username (or legacy optional display name)
	Requester string // authenticated account email
	Body      string
	CreatedAt time.Time

	Status     Status
	ConvID     string // shelley conversation driving this item
	Branch     string // git branch with the work
	Worktree   string // agent workspace directory (worktree or sandbox workspace, absolute)
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

// User is an account that can authenticate with a username or email address and password.
type User struct {
	ID           int64
	Username     string
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

// Visibility controls whether a worksheet is private or public.
type Visibility string

const (
	VisibilityPrivate Visibility = "private"
	VisibilityPublic  Visibility = "public"
)

// Permission is the access level granted to another user.
type Permission string

const (
	PermissionView Permission = "view"
	PermissionEdit Permission = "edit"
)

// WorksheetAccess stores ownership and sharing settings for one generated
// worksheet path.
type WorksheetAccess struct {
	Worksheet  string
	OwnerEmail string
	Visibility Visibility
	Finished   bool
	Shares     []WorksheetShare
	UpdatedAt  time.Time
}

// WorksheetShare grants another account view or edit rights.
type WorksheetShare struct {
	Worksheet  string
	Email      string
	Permission Permission
	CreatedAt  time.Time
	UpdatedAt  time.Time
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
  requester  TEXT NOT NULL DEFAULT '',
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
  username      TEXT NOT NULL UNIQUE,
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
CREATE INDEX IF NOT EXISTS auth_tokens_user ON auth_tokens(user_id, purpose);
CREATE TABLE IF NOT EXISTS worksheets (
  worksheet   TEXT PRIMARY KEY,
  owner_email TEXT NOT NULL COLLATE NOCASE,
  visibility  TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private','public')),
  created_at  INTEGER NOT NULL,
  updated_at  INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS worksheets_owner ON worksheets(owner_email);
CREATE TABLE IF NOT EXISTS worksheet_shares (
  worksheet  TEXT NOT NULL REFERENCES worksheets(worksheet) ON DELETE CASCADE,
  email      TEXT NOT NULL COLLATE NOCASE,
  permission TEXT NOT NULL CHECK (permission IN ('view','edit')),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY (worksheet, email)
);
CREATE INDEX IF NOT EXISTS worksheet_shares_email ON worksheet_shares(email);`
	if _, err := d.Exec(schema); err != nil {
		return err
	}
	// Databases created before the pipeline existed lack these columns.
	for _, col := range []string{
		"requester TEXT NOT NULL DEFAULT ''",
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
	if _, err := d.Exec("ALTER TABLE users ADD COLUMN username TEXT NOT NULL DEFAULT ''"); err != nil &&
		!strings.Contains(err.Error(), "duplicate column") {
		return fmt.Errorf("store: adding column username: %w", err)
	}
	if err := migrateUsernames(d); err != nil {
		return err
	}
	// Databases created before finished worksheets existed lack this column.
	if _, err := d.Exec("ALTER TABLE worksheets ADD COLUMN finished INTEGER NOT NULL DEFAULT 0"); err != nil &&
		!strings.Contains(err.Error(), "duplicate column") {
		return fmt.Errorf("store: adding column finished: %w", err)
	}
	return nil
}

func migrateUsernames(d *sql.DB) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for email, username := range map[string]string{
		"paul.hankin@pobox.com": "paulhankin",
		"g.n.hankin@gmail.com":  "gloriahankin",
	} {
		if _, err := tx.Exec(`UPDATE users SET username = ? WHERE email = ? AND username = ''`, username, email); err != nil {
			return fmt.Errorf("store: assigning username %s: %w", username, err)
		}
	}
	rows, err := tx.Query(`SELECT id FROM users WHERE username = '' ORDER BY id`)
	if err != nil {
		return err
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Close(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, err := tx.Exec(`UPDATE users SET username = ? WHERE id = ?`, fmt.Sprintf("user-%d", id), id); err != nil {
			return fmt.Errorf("store: assigning fallback username: %w", err)
		}
	}
	if _, err := tx.Exec(`CREATE UNIQUE INDEX IF NOT EXISTS users_username ON users(username)`); err != nil {
		return fmt.Errorf("store: indexing usernames: %w", err)
	}
	return tx.Commit()
}

// Add stores a new request and returns its ID.
func (db *DB) Add(r Request) (int64, error) {
	if r.Kind != KindNew && r.Kind != KindChange {
		return 0, fmt.Errorf("store: unknown kind %q", r.Kind)
	}
	now := time.Now().Unix()
	res, err := db.sql.Exec(
		`INSERT INTO requests (kind, worksheet, author, requester, body, created_at, status, updated_at)
		 VALUES (?,?,?,?,?,?,?,?)`,
		string(r.Kind), r.Worksheet, r.Author, r.Requester, r.Body, now, string(StatusQueued), now)
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

const selectRequests = `SELECT id, kind, worksheet, author, requester, body, created_at,
 status, conv_id, branch, worktree, note, preview, updated_at FROM requests`

func scanRequests(rows *sql.Rows) ([]Request, error) {
	var out []Request
	for rows.Next() {
		var r Request
		var created, updated int64
		var kind, status string
		var preview int
		if err := rows.Scan(&r.ID, &kind, &r.Worksheet, &r.Author, &r.Requester, &r.Body, &created,
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
func (db *DB) CreateUser(username, email string, passwordHash []byte) (User, error) {
	if username == "" {
		return User{}, fmt.Errorf("store: empty username")
	}
	now := time.Now().Unix()
	res, err := db.sql.Exec(
		`INSERT INTO users (username, email, password_hash, created_at, updated_at) VALUES (?,?,?,?,?)`,
		username, email, passwordHash, now, now)
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
	return db.scanUser(`SELECT id, username, email, password_hash, verified_at, created_at, updated_at FROM users WHERE id = ?`, id)
}

// UserByEmail returns an account by its case-insensitive email address.
func (db *DB) UserByEmail(email string) (User, error) {
	return db.scanUser(`SELECT id, username, email, password_hash, verified_at, created_at, updated_at FROM users WHERE email = ?`, email)
}

// UserByUsername returns an account by its unique username.
func (db *DB) UserByUsername(username string) (User, error) {
	return db.scanUser(`SELECT id, username, email, password_hash, verified_at, created_at, updated_at FROM users WHERE username = ?`, username)
}

func (db *DB) scanUser(query string, arg any) (User, error) {
	var u User
	var verified, created, updated int64
	err := db.sql.QueryRow(query, arg).Scan(
		&u.ID, &u.Username, &u.Email, &u.PasswordHash, &verified, &created, &updated)
	if err != nil {
		return User{}, err
	}
	u.Verified = verified != 0
	u.CreatedAt = time.Unix(created, 0)
	u.UpdatedAt = time.Unix(updated, 0)
	return u, nil
}

// Users returns all accounts ordered by username.
func (db *DB) Users() ([]User, error) {
	rows, err := db.sql.Query(`SELECT id, username, email, password_hash, verified_at, created_at, updated_at FROM users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var u User
		var verified, created, updated int64
		if err := rows.Scan(&u.ID, &u.Username, &u.Email, &u.PasswordHash, &verified, &created, &updated); err != nil {
			return nil, err
		}
		u.Verified = verified != 0
		u.CreatedAt = time.Unix(created, 0)
		u.UpdatedAt = time.Unix(updated, 0)
		users = append(users, u)
	}
	return users, rows.Err()
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

// EnsureWorksheets gives every generated worksheet a persistent owner and
// private visibility. Existing records are left unchanged.
func (db *DB) EnsureWorksheets(paths []string, defaultOwner string) error {
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().Unix()
	for _, path := range paths {
		if strings.TrimSpace(path) == "" {
			continue
		}
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO worksheets (worksheet, owner_email, visibility, created_at, updated_at)
			 VALUES (?,?,?,?,?)`, path, defaultOwner, string(VisibilityPrivate), now, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// WorksheetByPath returns the ownership and sharing settings for a worksheet.
func (db *DB) WorksheetByPath(path string) (WorksheetAccess, error) {
	var ws WorksheetAccess
	var visibility string
	var updated int64
	err := db.sql.QueryRow(
		`SELECT worksheet, owner_email, visibility, finished, updated_at FROM worksheets WHERE worksheet = ?`, path,
	).Scan(&ws.Worksheet, &ws.OwnerEmail, &visibility, &ws.Finished, &updated)
	if err != nil {
		return WorksheetAccess{}, err
	}
	ws.Visibility = Visibility(visibility)
	ws.UpdatedAt = time.Unix(updated, 0)
	ws.Shares, err = db.WorksheetShares(path)
	return ws, err
}

// WorksheetsOwnedBy returns all worksheet settings owned by an account.
func (db *DB) WorksheetsOwnedBy(email string) ([]WorksheetAccess, error) {
	rows, err := db.sql.Query(
		`SELECT worksheet, owner_email, visibility, finished, updated_at FROM worksheets WHERE owner_email = ? ORDER BY worksheet`, email)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorksheetAccess
	for rows.Next() {
		var ws WorksheetAccess
		var visibility string
		var updated int64
		if err := rows.Scan(&ws.Worksheet, &ws.OwnerEmail, &visibility, &ws.Finished, &updated); err != nil {
			return nil, err
		}
		ws.Visibility = Visibility(visibility)
		ws.UpdatedAt = time.Unix(updated, 0)
		ws.Shares, err = db.WorksheetShares(ws.Worksheet)
		if err != nil {
			return nil, err
		}
		out = append(out, ws)
	}
	return out, rows.Err()
}

// CanViewWorksheet reports whether an account may view a worksheet. Owners,
// users with an explicit share, and all signed-in users for public worksheets
// have access.
func (db *DB) CanViewWorksheet(path, email string) (bool, error) {
	var allowed bool
	err := db.sql.QueryRow(`
		SELECT EXISTS (
			SELECT 1
			FROM worksheets w
			WHERE w.worksheet = ?
			  AND (
				w.owner_email = ?
				OR w.visibility = 'public'
				OR EXISTS (
					SELECT 1 FROM worksheet_shares s
					WHERE s.worksheet = w.worksheet AND s.email = ?
				)
			)
		)`, path, email, email).Scan(&allowed)
	return allowed, err
}

// SetWorksheetVisibility changes a worksheet owned by owner to private or public.
func (db *DB) SetWorksheetVisibility(path, owner string, visibility Visibility) error {
	if visibility != VisibilityPrivate && visibility != VisibilityPublic {
		return fmt.Errorf("store: unknown visibility %q", visibility)
	}
	res, err := db.sql.Exec(
		`UPDATE worksheets SET visibility = ?, updated_at = ? WHERE worksheet = ? AND owner_email = ?`,
		string(visibility), time.Now().Unix(), path, owner)
	if err != nil {
		return err
	}
	return requireChanged(res, "worksheet not found or not owned by user")
}

// SetWorksheetShare creates or updates a private worksheet share.

// SetWorksheetFinished marks a worksheet owned by owner as finished or moves
// it back into the active list.
func (db *DB) SetWorksheetFinished(path, owner string, finished bool) error {
	res, err := db.sql.Exec(
		`UPDATE worksheets SET finished = ?, updated_at = ? WHERE worksheet = ? AND owner_email = ?`,
		finished, time.Now().Unix(), path, owner)
	if err != nil {
		return err
	}
	return requireChanged(res, "worksheet not found or not owned by user")
}
func (db *DB) SetWorksheetShare(path, owner, email string, permission Permission) error {
	if permission != PermissionView && permission != PermissionEdit {
		return fmt.Errorf("store: unknown permission %q", permission)
	}
	if strings.EqualFold(owner, email) {
		return fmt.Errorf("store: owner cannot share a worksheet with themselves")
	}
	tx, err := db.sql.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var visibility string
	if err := tx.QueryRow(
		`SELECT visibility FROM worksheets WHERE worksheet = ? AND owner_email = ?`, path, owner,
	).Scan(&visibility); err != nil {
		return err
	}
	if Visibility(visibility) != VisibilityPrivate {
		return fmt.Errorf("store: public worksheets cannot be shared")
	}
	now := time.Now().Unix()
	_, err = tx.Exec(
		`INSERT INTO worksheet_shares (worksheet, email, permission, created_at, updated_at)
		 VALUES (?,?,?,?,?)
		 ON CONFLICT(worksheet,email) DO UPDATE SET permission = excluded.permission, updated_at = excluded.updated_at`,
		path, email, string(permission), now, now)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// DeleteWorksheetShare removes one user's access from a worksheet.
func (db *DB) DeleteWorksheetShare(path, owner, email string) error {
	res, err := db.sql.Exec(
		`DELETE FROM worksheet_shares WHERE worksheet = ? AND email = ?
		 AND EXISTS (SELECT 1 FROM worksheets WHERE worksheet = ? AND owner_email = ?)`,
		path, email, path, owner)
	if err != nil {
		return err
	}
	return requireChanged(res, "share not found or worksheet not owned by user")
}

// WorksheetShares returns the grants for a worksheet.
func (db *DB) WorksheetShares(path string) ([]WorksheetShare, error) {
	rows, err := db.sql.Query(
		`SELECT worksheet, email, permission, created_at, updated_at
		 FROM worksheet_shares WHERE worksheet = ? ORDER BY email`, path)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WorksheetShare
	for rows.Next() {
		var share WorksheetShare
		var permission string
		var created, updated int64
		if err := rows.Scan(&share.Worksheet, &share.Email, &permission, &created, &updated); err != nil {
			return nil, err
		}
		share.Permission = Permission(permission)
		share.CreatedAt = time.Unix(created, 0)
		share.UpdatedAt = time.Unix(updated, 0)
		out = append(out, share)
	}
	return out, rows.Err()
}

func requireChanged(result sql.Result, message string) error {
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("store: %s", message)
	}
	return nil
}
