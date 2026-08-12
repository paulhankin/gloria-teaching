// Package store keeps worksheet requests and site settings in SQLite.
package store

import (
	"database/sql"
	"fmt"
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

// Request is a single wish submitted through the website.
type Request struct {
	ID        int64
	Kind      Kind
	Worksheet string // worksheet path ("math/venn_diagrams"), empty for KindNew
	Author    string // optional
	Body      string
	CreatedAt time.Time
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
  created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS requests_worksheet ON requests(worksheet);
CREATE TABLE IF NOT EXISTS settings (
  key   TEXT PRIMARY KEY,
  value TEXT NOT NULL
);`
	_, err := d.Exec(schema)
	return err
}

// Add stores a new request and returns its ID.
func (db *DB) Add(r Request) (int64, error) {
	if r.Kind != KindNew && r.Kind != KindChange {
		return 0, fmt.Errorf("store: unknown kind %q", r.Kind)
	}
	res, err := db.sql.Exec(
		`INSERT INTO requests (kind, worksheet, author, body, created_at) VALUES (?,?,?,?,?)`,
		string(r.Kind), r.Worksheet, r.Author, r.Body, time.Now().Unix())
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Delete removes a request.
func (db *DB) Delete(id int64) error {
	_, err := db.sql.Exec(`DELETE FROM requests WHERE id = ?`, id)
	return err
}

// All returns every request, newest first.
func (db *DB) All() ([]Request, error) {
	rows, err := db.sql.Query(
		`SELECT id, kind, worksheet, author, body, created_at FROM requests ORDER BY id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Request
	for rows.Next() {
		var r Request
		var ts int64
		var kind string
		if err := rows.Scan(&r.ID, &kind, &r.Worksheet, &r.Author, &r.Body, &ts); err != nil {
			return nil, err
		}
		r.Kind = Kind(kind)
		r.CreatedAt = time.Unix(ts, 0)
		out = append(out, r)
	}
	return out, rows.Err()
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
