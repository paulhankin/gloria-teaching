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
