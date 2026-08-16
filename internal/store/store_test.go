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

	u, err := db.CreateUser("person@example.com", []byte("hash"))
	if err != nil {
		t.Fatal(err)
	}
	if u.Verified || u.Email != "person@example.com" {
		t.Fatalf("new user = %#v", u)
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
