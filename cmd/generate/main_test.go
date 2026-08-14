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
	if err := os.MkdirAll(filepath.Join(build, "math", "new_sheet"), 0o755); err != nil {
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
	write(filepath.Join(build, "math", "new_sheet", "index.html"), "new")
	write(filepath.Join(build, site.ManifestName), `[{"subject":"math","name":"new_sheet"}]`)
	write(filepath.Join(target, "math", "old_sheet", "index.html"), "old")

	if err := publishOutput(build, target); err != nil {
		t.Fatal(err)
	}
	for path, want := range map[string]string{
		filepath.Join(target, "math", "new_sheet", "index.html"): "new",
		filepath.Join(target, "math", "old_sheet", "index.html"): "old",
	} {
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != want {
			t.Fatalf("%s = %q, want %q", path, got, want)
		}
	}
	worksheets, err := site.ReadManifest(filepath.Join(target, site.ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	if len(worksheets) != 1 || worksheets[0].Path() != "math/new_sheet" {
		t.Fatalf("published manifest = %#v", worksheets)
	}
}
