package worksheetrepo

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHasLocalUserDirectoriesFollowsSymlinks(t *testing.T) {
	root := t.TempDir()
	// An empty root has no user directories.
	if HasLocalUserDirectories(root) {
		t.Fatal("empty root reported user directories")
	}
	// A symlink to a directory counts (generate/<user> in production is a
	// symlink into /users/<user>).
	if err := os.MkdirAll(filepath.Join(root, "real-user"), 0o755); err != nil {
		t.Fatal(err)
	}
	linked := t.TempDir()
	if err := os.Symlink(filepath.Join(root, "real-user"), filepath.Join(linked, "link-user")); err != nil {
		t.Fatal(err)
	}
	if !HasLocalUserDirectories(linked) {
		t.Fatal("a symlink to a directory was not detected")
	}
	// A symlink to a FILE is not a user directory.
	file := filepath.Join(root, "notes.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	files := t.TempDir()
	if err := os.Symlink(file, filepath.Join(files, "file-link")); err != nil {
		t.Fatal(err)
	}
	if HasLocalUserDirectories(files) {
		t.Fatal("a symlink to a file was detected as a user directory")
	}
}

func TestLinkUserRepositories(t *testing.T) {
	root := t.TempDir()
	users := filepath.Join(root, "users")
	generate := filepath.Join(root, "generate")
	if err := os.MkdirAll(filepath.Join(users, "teacher"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := LinkUserRepositories(users, generate); err != nil {
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

func TestLinkUserRepositoriesReplacesManagedLink(t *testing.T) {
	root := t.TempDir()
	oldUsers := filepath.Join(root, "old")
	newUsers := filepath.Join(root, "new")
	generate := filepath.Join(root, "generate")
	for _, dir := range []string{filepath.Join(oldUsers, "teacher"), filepath.Join(newUsers, "teacher")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := LinkUserRepositories(oldUsers, generate); err != nil {
		t.Fatal(err)
	}
	if err := LinkUserRepositories(newUsers, generate); err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(filepath.Join(generate, "teacher"))
	if err != nil {
		t.Fatal(err)
	}
	want, _ := filepath.Abs(filepath.Join(newUsers, "teacher"))
	if got != want {
		t.Fatalf("replacement link target = %q, want %q", got, want)
	}
}

func TestPackagesNeedsOnlyWorksheetSource(t *testing.T) {
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

	got, err := Packages(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join("generate", "a-user", "fractions"),
		filepath.Join("generate", "z-user", "times"),
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Packages() = %q, want %q", got, want)
	}
}

func TestImportSourceUsesCoreModule(t *testing.T) {
	got, err := importSource([]string{filepath.Join("generate", "teacher", "fractions")})
	if err != nil {
		t.Fatal(err)
	}
	want := "\t_ \"learningmaterial/generate/teacher/fractions\"\n"
	if !contains(string(got), want) {
		t.Fatalf("generated imports = %q, want it to contain %q", got, want)
	}
}

func contains(s, part string) bool {
	for i := 0; i+len(part) <= len(s); i++ {
		if s[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
