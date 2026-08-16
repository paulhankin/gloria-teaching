package pipeline

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"learningmaterial/internal/store"
)

func TestPromptUsesUsername(t *testing.T) {
	p := &Pipeline{}
	got := p.prompt(store.Request{
		Kind: store.KindNew, Body: "Make a worksheet", Requester: "teacher@example.com", Author: "teacher",
	}, "")
	if !strings.Contains(got, "Requested by: teacher") || strings.Contains(got, "Requested by: teacher@example.com") {
		t.Fatalf("prompt requester attribution = %q", got)
	}
	if !strings.Contains(got, "generate/teacher/<worksheet>") {
		t.Fatalf("prompt source directory = %q", got)
	}
}

func TestAddWorkspaceCreatesNestedUserWorktree(t *testing.T) {
	root := t.TempDir()
	mainRepo := filepath.Join(root, "main")
	if err := os.MkdirAll(mainRepo, 0o755); err != nil {
		t.Fatal(err)
	}
	runGit := func(dir string, args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git -C %s %v: %v: %s", dir, args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	runGit(mainRepo, "init", "-b", "main")
	runGit(mainRepo, "config", "user.name", "Test")
	runGit(mainRepo, "config", "user.email", "test@example.com")
	if err := os.WriteFile(filepath.Join(mainRepo, "README.md"), []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainRepo, ".gitignore"), []byte("/generate/*/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(mainRepo, "add", ".")
	runGit(mainRepo, "commit", "-m", "Initial")

	p := &Pipeline{cfg: Config{
		Repo:          mainRepo,
		WorksheetRoot: filepath.Join(mainRepo, "generate"),
		WorkRoot:      filepath.Join(root, "work"),
	}}
	workspace := filepath.Join(p.cfg.WorkRoot, "req-1")
	if err := p.addWorkspace("teacher", "req-1", workspace); err != nil {
		t.Fatal(err)
	}
	userWorktree := filepath.Join(workspace, "generate", "teacher")
	if got := runGit(userWorktree, "branch", "--show-current"); got != "req-1" {
		t.Fatalf("nested branch = %q, want req-1", got)
	}
	if got := runGit(workspace, "status", "--porcelain"); got != "" {
		t.Fatalf("main worktree sees nested repository: %q", got)
	}
}

func TestRevisionsUsesWorksheetGitHistory(t *testing.T) {
	root := t.TempDir()
	repo := filepath.Join(root, "teacher")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
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
	path := filepath.Join(repo, "fractions")
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

	output := filepath.Join(root, "output")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "worksheets.json"), []byte(`[{"username":"teacher","subject":"math","name":"fractions"}]`), 0o644); err != nil {
		t.Fatal(err)
	}
	p := &Pipeline{cfg: Config{Repo: root, WorksheetRoot: root, OutputDir: output}}
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
