package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

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
		filepath.Join(root, "a-user", "fractions"),
		filepath.Join(root, "z-user", "times"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("worksheetPackages() = %q, want %q", got, want)
	}
}
