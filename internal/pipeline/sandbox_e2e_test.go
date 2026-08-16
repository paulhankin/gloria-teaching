package pipeline

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"learningmaterial/internal/store"
)

// TestSandboxedPipelineEndToEnd drives a full sandbox-mode request through
// the real pipeline with the builtin deterministic "predictable" model: the
// isolated shelley server runs with -predictable-only and every chat message
// is a "bash: <command>" instruction, which the predictable model executes
// verbatim through its bash tool inside the bubblewrap sandbox.
//
// Covered end to end: clone creation, sandboxed server start, a real agent
// turn editing BOTH clones (worksheet + core), gofmt/go build/make html
// inside the sandbox, commits in both repositories, the trusted host-side
// preview build, the commit import into the live repositories, the push to
// the (bare, local) remotes, the public rebuild, and the sandbox cleanup.
// No LLM traffic, no cost.
//
// The test needs the real repository layout: the pipeline builds previews
// with "go run ./cmd/..." inside the sandbox clone, so the live core
// repository must be the module this test runs in (the test skips itself
// when that checkout is unavailable or dirty in the wrong way).
func TestSandboxedPipelineEndToEnd(t *testing.T) {
	requireShelleySandbox(t)

	// The core repository is a temp clone of this module's own checkout: the
	// pipeline runs the Go generator in the sandbox clone, which needs the
	// real sources. Using a clone (not the live checkout) keeps the test
	// hermetic: imports and pushes land in temp repos only. Uncommitted
	// changes in the developer's working tree are simply not part of the
	// fixture (the clone uses committed state).
	sourceRepo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(sourceRepo, "go.mod")); err != nil {
		t.Skipf("test must run inside the learningmaterial checkout: %v", err)
	}

	root := t.TempDir()
	coreRepo := filepath.Join(root, "live-core")
	gitOK(t, root, "clone", "--no-local", sourceRepo, coreRepo)
	gitOK(t, coreRepo, "config", "user.name", "Test")
	gitOK(t, coreRepo, "config", "user.email", "test@example.com")
	// Detach-check out the source HEAD commit as local main so the clone
	// works even when the source checkout is on a detached or feature head.
	sourceHead := gitOK(t, sourceRepo, "rev-parse", "HEAD")
	gitOK(t, coreRepo, "checkout", "-B", "main", sourceHead)

	// Live worksheet repository for the requester, with one starter
	// worksheet so the generator has something to render.
	usersRoot := filepath.Join(root, "users")
	userRepo := filepath.Join(usersRoot, "teacher")
	initTestRepo(t, userRepo, "worksheets")
	commitFile(t, userRepo, "minimal/minimal.go", `// Package minimal builds a minimal test worksheet.
package minimal

import "learningmaterial/internal/sheet"

func init() {
	sheet.Register(sheet.Worksheet{
		Username: "teacher",
		Subject:  "math",
		Name:     "minimal",
		Title:    "Minimal",
		Build: func() *sheet.Doc {
			return &sheet.Doc{Title: "Minimal", Body: "<p>Aufgabe 1</p>", Solutions: "<p>Lösung 1</p>"}
		},
	})
}
`, "Add the minimal worksheet")

	// The live repositories need remotes: the pipeline pushes after the
	// import (cfg.Push), and the clone must not inherit anything writable.
	// The temp core clone already has "origin" pointing at the real
	// checkout; repoint it at the bare remote so a push can never touch the
	// developer's working tree.
	remoteBase := filepath.Join(root, "remotes")
	coreRemote := filepath.Join(remoteBase, "core.git")
	userRemote := filepath.Join(remoteBase, "teacher.git")
	for _, remote := range []string{coreRemote, userRemote} {
		if err := os.MkdirAll(remote, 0o755); err != nil {
			t.Fatal(err)
		}
		gitOK(t, remote, "init", "--bare")
	}
	gitOK(t, coreRepo, "remote", "set-url", "origin", coreRemote)
	gitOK(t, userRepo, "remote", "add", "origin", userRemote)

	// Request database.
	db, err := store.Open(filepath.Join(root, "requests.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// The predictable turn: edit BOTH clones, format, build, commit. The
	// predictable model executes a "bash: <command>" prompt verbatim through
	// its bash tool inside the sandbox. RawPrompt makes the pipeline pass
	// the request body through unwrapped (the predictable model
	// pattern-matches the whole prompt; the usual worksheet prompt wrapper
	// would hide the "bash: " prefix).
	//
	// The heredoc (the worksheet source) must come FIRST: inside a
	// "a && cat <<EOF ... EOF && b" chain the heredoc body would be attached
	// to the whole &&-list and the remaining commands would end up inside
	// the file. The "&&" chain follows after the heredoc's delimiter line.
	//
	// The nested worksheet repository is a real directory inside the clone
	// (no symlinks), so cmd/importworksheets picks it up directly (the
	// production symlink path is exercised by the worksheetrepo unit test).
	turnScript := `mkdir -p generate/teacher/subtraction && cat > generate/teacher/subtraction/subtraction.go <<'GO'
// Package subtraction builds a subtraction worksheet.
package subtraction

import "learningmaterial/internal/sheet"

func init() {
	sheet.Register(sheet.Worksheet{
		Username: "teacher",
		Subject:  "math",
		Name:     "subtraction",
		Title:    "Subtraktion",
		Build: func() *sheet.Doc {
			return &sheet.Doc{Title: "Subtraktion", Body: "<p>7 - 3 = ?</p>", Solutions: "<p>4</p>"}
		},
	})
}
GO
` + strings.Join([]string{
		// A small core change too, so BOTH repositories are ahead.
		`printf '\n// E2E marker: sandboxed predictable run.\n' >> README.md`,
		"gofmt -l -w .",
		"go build ./...",
		"make html",
		// Belt and braces for the pipeline's own host-side preview build: it
		// runs `go run ./cmd/generate`, which regenerates worksheets_local.go
		// via cmd/importworksheets. If the agent's build above already
		// prepared it this is a no-op; otherwise the turn itself produces a
		// tree the trusted preview can build. (The worksheet clone is a real
		// directory, so Prepare discovers it without any symlink handling.)
		"go run ./cmd/importworksheets",
		// The exe.dev bash wrapper rejects blind "git add -A": name the
		// files explicitly, exactly as AGENTS.md discipline expects.
		// worksheets_local.go and output/ are gitignored (regenerated by
		// every build), so the core commit carries only the README marker.
		"git -C generate/teacher add subtraction/subtraction.go",
		`git -C generate/teacher commit -qm "Add the subtraction worksheet"`,
		"git add README.md",
		`git commit -qm "Register the subtraction worksheet"`,
	}, " && ")

	reqID, err := db.Add(store.Request{Kind: store.KindNew, Author: "teacher", Body: "bash: " + turnScript})
	if err != nil {
		t.Fatal(err)
	}

	p := New(db, Config{
		Repo:                   coreRepo,
		WorksheetRoot:          usersRoot,
		WorksheetRemoteBaseURL: remoteBase,
		PreviewRoot:            filepath.Join(root, "previews"),
		OutputDir:              filepath.Join(root, "output"),
		Push:                   true,
		Sandbox:                true,
		SandboxRoot:            filepath.Join(root, "sandboxes"),
		ShelleyPredictableOnly: true,
		ChatExtraArgs:          []string{"-model", "predictable"},
		RawPrompt:              true,
		JanitorInterval:        -1,
	})
	p.db.EnsureWorksheets([]string{"math/minimal"}, "teacher")

	it, err := db.Get(reqID)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.run(it, ""); err != nil {
		// Keep the sandbox for inspection on failure.
		paths, _ := sandboxPathsFor(p.cfg.SandboxRoot, reqID, "teacher")
		t.Logf("retained sandbox: %s", paths.Root)
		if data, readErr := os.ReadFile(paths.ServerLog); readErr == nil {
			t.Logf("server.log tail:\n%s", tail(string(data), 1500))
		}
		// Dump the worksheet source the turn produced: a mangled file here
		// means the bash tool received a corrupted command.
		broken, _ := os.ReadFile(filepath.Join(paths.Workspace, "generate", "teacher", "subtraction", "subtraction.go"))
		t.Logf("subtraction.go as written (escaped):\n%q", string(broken))
		// Dump the bash tool's result: the first clue for a command that
		// never ran (e.g. a rejected working directory). The server is
		// stopped by now, so a direct read-only sqlite query is safe.
		out, qerr := exec.Command("sqlite3", "-readonly", paths.DB,
			"select json_extract(llm_data,'$.Content[0].ToolResult[0].Text') from messages"+
				" where type='user' and json_extract(llm_data,'$.Content[0].ToolUseID') != ''").CombinedOutput()
		t.Logf("bash tool results (sqlite, err=%v):\n%s", qerr, tail(string(out), 2000))
		t.Fatalf("run: %v", err)
	}

	// The request is done and the sandbox was cleaned up.
	got, err := db.Get(reqID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != store.StatusDone {
		t.Fatalf("request status = %q (%q), want done", got.Status, got.Note)
	}
	paths, err := sandboxPathsFor(p.cfg.SandboxRoot, reqID, "teacher")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(paths.Root); !os.IsNotExist(err) {
		t.Errorf("the sandbox directory survived publication")
	}

	// The commits reached both live repositories and were pushed.
	if got := gitOK(t, userRepo, "show", "main:subtraction/subtraction.go"); !strings.Contains(got, "package subtraction") {
		t.Errorf("the worksheet repository is missing the agent's worksheet: %q", got)
	}
	if got := gitOK(t, coreRepo, "show", "main:README.md"); !strings.Contains(got, "E2E marker") {
		t.Errorf("the core repository is missing the agent's core change")
	}
	if got := gitOK(t, userRemote, "show", "main:subtraction/subtraction.go"); !strings.Contains(got, "package subtraction") {
		t.Errorf("the worksheet remote is missing the push")
	}
	if got := gitOK(t, coreRemote, "show", "main:README.md"); !strings.Contains(got, "E2E marker") {
		t.Errorf("the core remote is missing the push")
	}

	// The trusted preview build ran during the request (HasPreview was set)
	// and its directory was removed again by the publication cleanup — the
	// preview is ephemeral by design. The public output keeps the new
	// worksheet.
	if got.HasPreview {
		t.Error("HasPreview is still set after the publication cleanup")
	}
	if _, err := os.Stat(p.previewDir(reqID)); !os.IsNotExist(err) {
		t.Errorf("the preview directory survived the cleanup")
	}
	for _, base := range []string{p.cfg.OutputDir} {
		if _, err := os.Stat(filepath.Join(base, "teacher", "subtraction", "index.html")); err != nil {
			t.Errorf("%s is missing the generated worksheet: %v", base, err)
		}
	}

	// Both clones imported cleanly; the live checkouts are pristine.
	if out := strings.TrimSpace(gitOK(t, coreRepo, "status", "--porcelain")); out != "" {
		t.Errorf("the live core checkout is dirty after the import: %s", out)
	}
	if out := strings.TrimSpace(gitOK(t, userRepo, "status", "--porcelain")); out != "" {
		t.Errorf("the live worksheet checkout is dirty after the import: %s", out)
	}
}
