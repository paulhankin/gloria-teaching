package pipeline

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"learningmaterial/internal/store"
)

func TestPromptUsesAuthenticatedRequester(t *testing.T) {
	p := &Pipeline{}
	got := p.prompt(store.Request{
		Kind: store.KindNew, Body: "Make a worksheet", Requester: "teacher@example.com", Author: "Old name",
	}, "")
	if !strings.Contains(got, "Requested by: teacher@example.com") || strings.Contains(got, "Requested by: Old name") {
		t.Fatalf("prompt requester attribution = %q", got)
	}
}

func TestRevisionsUsesWorksheetGitHistory(t *testing.T) {
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}
	git("init", "-b", "main")
	git("config", "user.name", "Test")
	git("config", "user.email", "test@example.com")
	path := filepath.Join(repo, "generate", "math", "fractions")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(body, message string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(path, "sheet.go"), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		git("add", ".")
		git("commit", "-m", message)
	}
	write("first", "First worksheet version")
	write("second", "Second worksheet version")

	p := &Pipeline{cfg: Config{Repo: repo}}
	revisions, err := p.Revisions("math/fractions")
	if err != nil {
		t.Fatal(err)
	}
	if len(revisions) != 2 || !revisions[0].Current || revisions[1].Current {
		t.Fatalf("revisions = %#v", revisions)
	}
	if revisions[0].Subject != "Second worksheet version" || revisions[1].Subject != "First worksheet version" {
		t.Fatalf("revision order = %#v", revisions)
	}
}
