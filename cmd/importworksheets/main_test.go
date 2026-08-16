package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLinkUserRepositories(t *testing.T) {
	root := t.TempDir()
	users := filepath.Join(root, "users")
	generate := filepath.Join(root, "generate")
	if err := os.MkdirAll(filepath.Join(users, "teacher"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := linkUserRepositories(users, generate); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(filepath.Join(generate, "teacher"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(filepath.Join(users, "teacher"))
	if got != want {
		t.Fatalf("link target = %q, want %q", got, want)
	}
}

func TestWorksheetPackages(t *testing.T) {
	root := t.TempDir()
	write := func(path string) {
		t.Helper()
		path = filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("package sheet"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("z-user/times/sheet.go")
	write("a-user/fractions/sheet.go")
	write("a-user/fractions/sheet_test.go")
	write("a-user/tests-only/sheet_test.go")
	write(".hidden/ignored/sheet.go")

	got, err := worksheetPackages(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join("generate", "a-user", "fractions"),
		filepath.Join("generate", "z-user", "times"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("worksheetPackages() = %q, want %q", got, want)
	}
}
