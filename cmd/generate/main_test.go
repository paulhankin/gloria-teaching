package main

import (
	"os"
	"path/filepath"
	"testing"

	"learningmaterial/internal/site"
)

func TestWorksheetVersionChangesWithGeneratedContent(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "math", "fractions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name+".pdf"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("index", "worksheet v1")
	write("solutions", "solutions v1")
	v1, err := worksheetVersion(root, "math/fractions")
	if err != nil {
		t.Fatal(err)
	}
	write("index", "worksheet v2")
	v2, err := worksheetVersion(root, "math/fractions")
	if err != nil {
		t.Fatal(err)
	}
	if v1 == v2 {
		t.Fatalf("version did not change: %q", v1)
	}
	if len(v1) != 12 || len(v2) != 12 {
		t.Fatalf("versions should be short hashes: %q, %q", v1, v2)
	}
}

func TestPublishOutput(t *testing.T) {
	root := t.TempDir()
	build := filepath.Join(root, "build")
	target := filepath.Join(root, "output")
	if err := os.MkdirAll(filepath.Join(build, "teacher", "new_sheet"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(target, "math", "old_sheet"), 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, body string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(filepath.Join(build, "teacher", "new_sheet", "index.html"), "new")
	write(filepath.Join(build, site.ManifestName), `[{"username":"teacher","subject":"math","name":"new_sheet"}]`)
	write(filepath.Join(target, "math", "old_sheet", "index.html"), "old")

	if err := publishOutput(build, target); err != nil {
		t.Fatal(err)
	}
	worksheets, err := site.ReadManifest(filepath.Join(target, site.ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if err := pruneOutput(target, worksheets); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(target, "teacher", "new_sheet", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Fatalf("published worksheet = %q, want new", got)
	}
	if _, err := os.Stat(filepath.Join(target, "math", "old_sheet")); !os.IsNotExist(err) {
		t.Fatalf("stale worksheet was not removed: %v", err)
	}
	if len(worksheets) != 1 || worksheets[0].Path() != "math/new_sheet" {
		t.Fatalf("published manifest = %#v", worksheets)
	}
}
