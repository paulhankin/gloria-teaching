package main

import (
	"path/filepath"
	"testing"

	"learningmaterial/internal/site"
	"learningmaterial/internal/store"
)

func TestCanServeWorksheetUsesGeneratedUsernamePath(t *testing.T) {
	root := t.TempDir()
	testDB, err := store.Open(filepath.Join(root, "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()
	manifest := filepath.Join(root, site.ManifestName)
	if err := site.WriteManifest(manifest, []site.Worksheet{{Username: "teacher", Subject: "math", Name: "fractions"}}); err != nil {
		t.Fatal(err)
	}
	if err := testDB.EnsureWorksheets([]string{"math/fractions"}, "teacher@example.com"); err != nil {
		t.Fatal(err)
	}

	oldDB, oldManifest := db, manifestPath
	db, manifestPath = testDB, manifest
	defer func() { db, manifestPath = oldDB, oldManifest }()

	if !canServeWorksheet("/teacher/fractions/index.pdf", "teacher@example.com") {
		t.Fatal("generated username path was denied")
	}
	if canServeWorksheet("/math/fractions/index.pdf", "teacher@example.com") {
		t.Fatal("old subject path was still accepted")
	}
}

func TestPublicWorksheetsFiltersByUsernameAndVisibility(t *testing.T) {
	testDB, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer testDB.Close()

	oldDB := db
	db = testDB
	defer func() { db = oldDB }()

	manifest := []site.Worksheet{
		{Username: "teacher", Subject: "math", Name: "public"},
		{Username: "teacher", Subject: "math", Name: "private"},
		{Username: "other", Subject: "math", Name: "other"},
	}
	paths := []string{"math/public", "math/private", "math/other"}
	if err := db.EnsureWorksheets(paths, "teacher@example.com"); err != nil {
		t.Fatal(err)
	}
	if err := db.SetWorksheetVisibility("math/public", "teacher@example.com", store.VisibilityPublic); err != nil {
		t.Fatal(err)
	}
	if err := db.SetWorksheetVisibility("math/other", "teacher@example.com", store.VisibilityPublic); err != nil {
		t.Fatal(err)
	}

	got, err := publicWorksheets(manifest, "teacher", "teacher@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Name != "public" {
		t.Fatalf("publicWorksheets() = %#v", got)
	}
}

func TestWorksheetForUserRequiresUsernameAndName(t *testing.T) {
	manifest := []site.Worksheet{
		{Username: "teacher", Subject: "math", Name: "fractions"},
		{Username: "other", Subject: "language", Name: "fractions"},
	}
	got, ok := worksheetForUser(manifest, "other", "fractions")
	if !ok || got.Subject != "language" {
		t.Fatalf("worksheetForUser() = %#v, %v", got, ok)
	}
	if _, ok := worksheetForUser(manifest, "teacher", "missing"); ok {
		t.Fatal("worksheetForUser found a missing worksheet")
	}
}
