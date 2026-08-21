package store

import (
	"bytes"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestUserAndAuthTokenLifecycle(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	u, err := db.CreateUser("person-1", "person@example.com", []byte("hash"))
	if err != nil {
		t.Fatal(err)
	}
	if u.Verified || u.Username != "person-1" || u.Email != "person@example.com" {
		t.Fatalf("new user = %#v", u)
	}
	byUsername, err := db.UserByUsername("person-1")
	if err != nil || byUsername.ID != u.ID {
		t.Fatalf("username lookup = %#v, %v", byUsername, err)
	}
	byEmail, err := db.UserByEmail("PERSON@example.com")
	if err != nil || byEmail.ID != u.ID {
		t.Fatalf("case-insensitive lookup = %#v, %v", byEmail, err)
	}
	if err := db.MarkUserVerified(u.ID); err != nil {
		t.Fatal(err)
	}
	verified, err := db.UserByID(u.ID)
	if err != nil || !verified.Verified {
		t.Fatalf("verified user = %#v, %v", verified, err)
	}

	token := []byte("token hash")
	if err := db.PutAuthToken(u.ID, "reset", token, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, err := db.ConsumeAuthToken(token, "reset")
	if err != nil || got.UserID != u.ID {
		t.Fatalf("consumed token = %#v, %v", got, err)
	}
	if _, err := db.ConsumeAuthToken(token, "reset"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("second token use error = %v", err)
	}

	newHash := []byte("new hash")
	if err := db.SetUserPassword(u.ID, newHash); err != nil {
		t.Fatal(err)
	}
	updated, err := db.UserByID(u.ID)
	if err != nil || !bytes.Equal(updated.PasswordHash, newHash) {
		t.Fatalf("updated user = %#v, %v", updated, err)
	}
}

func TestWorksheetOwnershipVisibilityAndSharing(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	paths := []string{"math/fractions", "math/times"}
	if err := db.EnsureWorksheets(paths, "g.n.hankin@gmail.com"); err != nil {
		t.Fatal(err)
	}
	// Reconciliation must not replace an existing owner.
	if err := db.EnsureWorksheets(paths, "someone@example.com"); err != nil {
		t.Fatal(err)
	}
	owned, err := db.WorksheetsOwnedBy("G.N.HANKIN@gmail.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(owned) != 2 || owned[0].OwnerEmail != "g.n.hankin@gmail.com" || owned[0].Visibility != VisibilityPrivate {
		t.Fatalf("owned worksheets = %#v", owned)
	}

	if err := db.SetWorksheetShare("math/fractions", "g.n.hankin@gmail.com", "friend@example.com", PermissionView); err != nil {
		t.Fatal(err)
	}
	if err := db.SetWorksheetShare("math/fractions", "g.n.hankin@gmail.com", "friend@example.com", PermissionEdit); err != nil {
		t.Fatal(err)
	}
	ws, err := db.WorksheetByPath("math/fractions")
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Shares) != 1 || ws.Shares[0].Permission != PermissionEdit {
		t.Fatalf("shares = %#v", ws.Shares)
	}

	if err := db.SetWorksheetVisibility("math/fractions", "g.n.hankin@gmail.com", VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	if err := db.SetWorksheetShare("math/fractions", "g.n.hankin@gmail.com", "other@example.com", PermissionView); err == nil {
		t.Fatal("sharing a public worksheet succeeded")
	}
	if err := db.SetWorksheetVisibility("math/fractions", "not-owner@example.com", VisibilityPrivate); err == nil {
		t.Fatal("non-owner changed worksheet visibility")
	}
	if err := db.DeleteWorksheetShare("math/fractions", "g.n.hankin@gmail.com", "friend@example.com"); err != nil {
		t.Fatal(err)
	}
}

func TestMigrateAssignsExistingUsernames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.db")
	legacy, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = legacy.Exec(`CREATE TABLE users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		email TEXT NOT NULL COLLATE NOCASE UNIQUE,
		password_hash BLOB NOT NULL,
		verified_at INTEGER NOT NULL DEFAULT 0,
		created_at INTEGER NOT NULL,
		updated_at INTEGER NOT NULL
	);
	INSERT INTO users (email, password_hash, created_at, updated_at) VALUES
		('paul.hankin@pobox.com', X'00', 1, 1),
		('g.n.hankin@gmail.com', X'00', 1, 1),
		('other@example.com', X'00', 1, 1);`)
	if err != nil {
		t.Fatal(err)
	}
	if err := legacy.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for email, want := range map[string]string{
		"paul.hankin@pobox.com": "paulhankin",
		"g.n.hankin@gmail.com":  "gloriahankin",
		"other@example.com":     "user-3",
	} {
		user, err := db.UserByEmail(email)
		if err != nil || user.Username != want {
			t.Errorf("username for %s = %q, %v; want %q", email, user.Username, err, want)
		}
	}
	if _, err := db.CreateUser("paulhankin", "duplicate@example.com", []byte("hash")); err == nil {
		t.Fatal("duplicate username was accepted")
	}
}

func TestTagTreeAndWorksheetAssignment(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.EnsureWorksheets([]string{"math/venn", "math/prices"}, "owner@example.com"); err != nil {
		t.Fatal(err)
	}

	grade, err := db.CreateTag("owner@example.com", "First Grade", 0)
	if err != nil {
		t.Fatal(err)
	}
	maths, err := db.CreateTag("owner@example.com", "Mathematics", grade)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTag("owner@example.com", "Music", grade); err != nil {
		t.Fatal(err)
	}
	if _, err := db.CreateTag("owner@example.com", "  ", 0); err == nil {
		t.Fatal("empty tag name accepted")
	}
	if _, err := db.CreateTag("owner@example.com", "Orphan", 9999); err == nil {
		t.Fatal("tag with unknown parent accepted")
	}

	tags, err := db.Tags("owner@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(tags) != 1 || tags[0].Name != "First Grade" || len(tags[0].Children) != 2 {
		t.Fatalf("tag tree = %#v", tags)
	}
	if tags[0].Children[0].Name != "Mathematics" || tags[0].Children[1].Name != "Music" {
		t.Fatalf("children = %#v", tags[0].Children)
	}

	if err := db.RenameTag(maths, "owner@example.com", "Maths"); err != nil {
		t.Fatal(err)
	}
	if err := db.RenameTag(maths, "other@example.com", "Nope"); err == nil {
		t.Fatal("foreign owner renamed a tag")
	}

	if err := db.SetWorksheetTags("math/venn", "owner@example.com", []int64{maths}); err != nil {
		t.Fatal(err)
	}
	ids, err := db.WorksheetTagIDs("math/venn")
	if err != nil || !ids[maths] || len(ids) != 1 {
		t.Fatalf("worksheet tags = %v, %v", ids, err)
	}
	if err := db.SetWorksheetTags("math/venn", "other@example.com", nil); err == nil {
		t.Fatal("foreign owner tagged a worksheet")
	}

	// Deleting the parent cascades to children and assignments.
	if err := db.DeleteTag(grade, "owner@example.com"); err != nil {
		t.Fatal(err)
	}
	tags, err = db.Tags("owner@example.com")
	if err != nil || len(tags) != 0 {
		t.Fatalf("after delete tags = %#v, %v", tags, err)
	}
	ids, err = db.WorksheetTagIDs("math/venn")
	if err != nil || len(ids) != 0 {
		t.Fatalf("after delete worksheet tags = %v, %v", ids, err)
	}
}

func TestCanViewWorksheet(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.EnsureWorksheets([]string{"math/private", "math/public"}, "owner@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetWorksheetShare("math/private", "owner@example.com", "friend@example.com", PermissionView); err != nil {
		t.Fatal(err)
	}
	if err := db.SetWorksheetVisibility("math/public", "owner@example.com", VisibilityPublic); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		path, email string
		want        bool
	}{
		{"math/private", "owner@example.com", true},
		{"math/private", "OWNER@example.com", true},
		{"math/private", "friend@example.com", true},
		{"math/private", "stranger@example.com", false},
		{"math/public", "stranger@example.com", true},
		{"math/missing", "owner@example.com", false},
	}
	for _, tt := range tests {
		got, err := db.CanViewWorksheet(tt.path, tt.email)
		if err != nil {
			t.Errorf("CanViewWorksheet(%q, %q): %v", tt.path, tt.email, err)
		} else if got != tt.want {
			t.Errorf("CanViewWorksheet(%q, %q) = %v, want %v", tt.path, tt.email, got, tt.want)
		}
	}
}
