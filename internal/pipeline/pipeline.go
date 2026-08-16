// Package pipeline turns worksheet requests into work items and drives them
// through the agent: queued -> working -> review -> done/rejected.
//
// Every work item is developed in its own git worktree on its own branch, so
// items in different lanes (one lane per worksheet, plus one lane per request
// for a brand new worksheet) can be worked on independently. Within a lane the
// work is strictly sequential: a queued item only starts once the previous item
// of that lane has been approved or rejected.
package pipeline

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"learningmaterial/internal/site"
	"learningmaterial/internal/store"
)

// Config describes where the pipeline works.
type Config struct {
	Repo        string // main git checkout (the served one)
	WorkRoot    string // parent directory for the per-item worktrees
	PreviewRoot string // parent directory for the per-item preview builds
	OutputDir   string // the directory cmd/serve serves
	Push        bool   // push to origin after a merge
}

// Pipeline runs the work items.
type Pipeline struct {
	cfg Config
	db  *store.DB

	mu        sync.Mutex
	busy      map[string]bool // lane -> a goroutine is working on it
	publishMu sync.Mutex      // serializes main merges and output publication
	log       []string        // recent pipeline events, newest last

	wake chan struct{}
}

// New creates a pipeline. Start must be called to run it.
func New(db *store.DB, cfg Config) *Pipeline {
	return &Pipeline{cfg: cfg, db: db, busy: map[string]bool{}, wake: make(chan struct{}, 1)}
}

// Start begins the scheduler loop.
func (p *Pipeline) Start() {
	ready := p.recover()
	for _, it := range ready {
		p.publishAsync(it)
	}
	go p.loop()
}

// Kick asks the scheduler to look for work now.
func (p *Pipeline) Kick() {
	select {
	case p.wake <- struct{}{}:
	default:
	}
}

// Logf records a pipeline event (also visible in the admin UI).
func (p *Pipeline) Logf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("pipeline: %s", msg)
	p.mu.Lock()
	defer p.mu.Unlock()
	p.log = append(p.log, time.Now().Format("15:04:05")+"  "+msg)
	if len(p.log) > 60 {
		p.log = p.log[len(p.log)-60:]
	}
}

// Log returns the recent pipeline events, newest first.
func (p *Pipeline) Log() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]string, 0, len(p.log))
	for i := len(p.log) - 1; i >= 0; i-- {
		out = append(out, p.log[i])
	}
	return out
}

// recover puts interrupted agent work back in the queue and returns completed
// items that were waiting under the old manual-approval workflow.
func (p *Pipeline) recover() []store.Request {
	items, err := p.db.Active()
	if err != nil {
		log.Printf("pipeline: recover: %v", err)
		return nil
	}
	var ready []store.Request
	for _, it := range items {
		switch it.Status {
		case store.StatusWorking:
			if it.HasPreview && it.Branch != "" {
				ready = append(ready, it)
			} else {
				p.db.SetStatus(it.ID, store.StatusQueued, "requeued after a server restart")
			}
		case store.StatusReview:
			ready = append(ready, it)
		}
	}
	return ready
}

func (p *Pipeline) loop() {
	tick := time.NewTicker(5 * time.Second)
	defer tick.Stop()
	for {
		p.schedule()
		select {
		case <-tick.C:
		case <-p.wake:
		}
	}
}

// schedule starts the next queued item of every free lane.
func (p *Pipeline) schedule() {
	items, err := p.db.Active()
	if err != nil {
		log.Printf("pipeline: schedule: %v", err)
		return
	}
	blocked := map[string]bool{} // lane -> an item is in flight or waiting for a decision
	for _, it := range items {
		if it.Status != store.StatusQueued {
			blocked[it.Lane()] = true
		}
	}
	p.mu.Lock()
	for lane := range p.busy {
		blocked[lane] = true
	}
	p.mu.Unlock()

	for _, it := range items {
		if it.Status != store.StatusQueued || blocked[it.Lane()] {
			continue
		}
		blocked[it.Lane()] = true
		p.start(it, "")
	}
}

// start runs one work item in the background. followUp is a refinement message
// for an existing conversation ("" starts a fresh one).
func (p *Pipeline) start(it store.Request, followUp string) {
	lane := it.Lane()
	p.mu.Lock()
	if p.busy[lane] {
		p.mu.Unlock()
		return
	}
	p.busy[lane] = true
	p.mu.Unlock()

	p.db.SetPreview(it.ID, false)
	p.db.SetStatus(it.ID, store.StatusWorking, "the agent is working on it")
	go func() {
		defer func() {
			p.mu.Lock()
			delete(p.busy, lane)
			p.mu.Unlock()
			p.Kick()
		}()
		if err := p.run(it, followUp); err != nil {
			p.Logf("#%d failed: %v", it.ID, err)
			p.db.SetStatus(it.ID, store.StatusFailed, err.Error())
			return
		}
	}()
}

// run does the actual work for one item: worktree, agent, commit, preview.
func (p *Pipeline) run(it store.Request, followUp string) error {
	cur, err := p.db.Get(it.ID)
	if err != nil {
		return err
	}
	it = cur

	// A restart may have interrupted publication after the branch was already
	// merged. In that case, finish rebuilding instead of asking the agent to
	// redo work that is already on main.
	if it.Branch != "" {
		merged, err := p.isAncestor(it.Branch, "main")
		if err == nil && merged {
			return p.publish(it, it.Note)
		}
	}

	worktree := it.Worktree
	branch := it.Branch
	if worktree == "" {
		branch = fmt.Sprintf("req-%d", it.ID)
		worktree = filepath.Join(p.cfg.WorkRoot, branch)
		if err := p.addWorktree(branch, worktree); err != nil {
			return err
		}
		if err := p.db.SetRun(it.ID, "", branch, worktree); err != nil {
			return err
		}
	}

	convID := it.ConvID
	prompt := p.prompt(it, followUp)
	if convID == "" {
		p.Logf("#%d: starting the agent in %s", it.ID, worktree)
		convID, err = p.chatNew(worktree, prompt)
		if err != nil {
			return err
		}
		if err := p.db.SetRun(it.ID, convID, branch, worktree); err != nil {
			return err
		}
	} else {
		if followUp == "" {
			// Resuming after a server restart.
			prompt = "The server restarted while you were working. " +
				"Check the state of the worktree, finish the work, commit it " +
				"and summarise what you changed."
		}
		p.Logf("#%d: sending a refinement to the agent", it.ID)
		if err := p.chatContinue(convID, prompt); err != nil {
			return err
		}
	}

	summary, err := p.waitForTurn(convID)
	if err != nil {
		return err
	}

	if err := p.commitLeftovers(worktree, it); err != nil {
		return err
	}
	changed, err := p.hasCommits(branch)
	if err != nil {
		return err
	}
	if !changed {
		return fmt.Errorf("the agent did not commit anything: %s", firstLine(summary))
	}

	p.Logf("#%d: validating the generated worksheet", it.ID)
	if err := p.buildPreview(it, worktree); err != nil {
		return fmt.Errorf("worksheet build: %w", err)
	}
	p.db.SetPreview(it.ID, true)
	p.Logf("#%d: finished; publishing automatically", it.ID)
	return p.publish(it, summary)
}

func firstLine(s string) string {
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		return s[:i]
	}
	return s
}

// prompt builds the instruction for the agent.
func (p *Pipeline) prompt(it store.Request, followUp string) string {
	var b strings.Builder
	if followUp != "" {
		b.WriteString("A refinement was requested for the work you just did:\n\n")
		b.WriteString(followUp)
		b.WriteString("\n\nAdjust your work accordingly, rebuild (`gofmt -l -w .`, `go build ./...`, " +
			"`make html`) and amend or add a commit. Do not push.\n")
		return b.String()
	}
	b.WriteString("You are working on the `learningmaterial` repository in this worktree. " +
		"Follow AGENTS.md.\n\n")
	if it.Kind == store.KindChange {
		fmt.Fprintf(&b, "A change was requested for the worksheet `%s` "+
			"(directory `generate/%s`):\n\n", it.Worksheet, it.Worksheet)
	} else {
		b.WriteString("A new worksheet was requested:\n\n")
	}
	b.WriteString(it.Body)
	b.WriteString("\n\n")
	requester := it.Author
	if requester == "" {
		requester = it.Requester
	}
	if requester != "" {
		fmt.Fprintf(&b, "Requested by: %s\n\n", requester)
	}
	b.WriteString("Implement it. Then run `gofmt -l -w .`, `go build ./...` and `make html`, " +
		"and commit everything on the current branch with a good commit message. " +
		"Do NOT push and do NOT switch branches. " +
		"Finish with a two or three sentence summary of what you changed.\n")
	return b.String()
}

// --- git -------------------------------------------------------------------

func (p *Pipeline) git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %v: %s",
			strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func (p *Pipeline) addWorktree(branch, dir string) error {
	if err := os.MkdirAll(p.cfg.WorkRoot, 0o755); err != nil {
		return err
	}
	os.RemoveAll(dir)
	p.git(p.cfg.Repo, "worktree", "prune")
	p.git(p.cfg.Repo, "branch", "-D", branch)
	_, err := p.git(p.cfg.Repo, "worktree", "add", "-b", branch, dir, "main")
	return err
}

func (p *Pipeline) removeWorktree(it store.Request) {
	if it.Worktree != "" {
		p.git(p.cfg.Repo, "worktree", "remove", "--force", it.Worktree)
		os.RemoveAll(it.Worktree)
	}
	if it.Branch != "" {
		p.git(p.cfg.Repo, "branch", "-D", it.Branch)
	}
	os.RemoveAll(p.previewDir(it.ID))
	p.db.SetPreview(it.ID, false)
}

// commitLeftovers commits anything the agent forgot to commit.
func (p *Pipeline) commitLeftovers(worktree string, it store.Request) error {
	out, err := p.git(worktree, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) == "" {
		return nil
	}
	if _, err := p.git(worktree, "add", "-A"); err != nil {
		return err
	}
	msg := fmt.Sprintf("Uncommitted leftovers of request #%d", it.ID)
	_, err = p.git(worktree, "commit", "-m", msg)
	return err
}

// hasCommits reports whether the branch is ahead of main.
func (p *Pipeline) hasCommits(branch string) (bool, error) {
	out, err := p.git(p.cfg.Repo, "rev-list", "--count", "main.."+branch)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "0", nil
}

func (p *Pipeline) isAncestor(ancestor, descendant string) (bool, error) {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", ancestor, descendant)
	cmd.Dir = p.cfg.Repo
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
		return false, nil
	}
	return false, err
}

// --- shelley ---------------------------------------------------------------

func (p *Pipeline) chatNew(cwd, prompt string) (string, error) {
	cmd := exec.Command("shelley", "client", "chat", "-cwd", cwd, "-p", prompt)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("shelley client chat: %v", err)
	}
	var res struct {
		ConversationID string `json:"conversation_id"`
	}
	if err := json.Unmarshal(out, &res); err != nil || res.ConversationID == "" {
		return "", fmt.Errorf("shelley client chat: unexpected output %q", string(out))
	}
	return res.ConversationID, nil
}

func (p *Pipeline) chatContinue(convID, prompt string) error {
	cmd := exec.Command("shelley", "client", "chat", "-c", convID, "-p", prompt)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("shelley client chat -c: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// waitForTurn blocks until the agent turn ends and returns its last message.
// It polls `shelley client list` instead of using `read -wait`, which does not
// reliably return when it attaches to a turn that is already running.
func (p *Pipeline) waitForTurn(convID string) (string, error) {
	deadline := time.Now().Add(90 * time.Minute)
	idle := 0
	for time.Now().Before(deadline) {
		time.Sleep(5 * time.Second)
		working, err := p.working(convID)
		if err != nil {
			return "", err
		}
		if working {
			idle = 0
			continue
		}
		// Two idle polls in a row: the turn really has ended (right after
		// `chat` the conversation is not marked as working yet).
		idle++
		if idle >= 2 {
			return p.lastAgentMessage(convID)
		}
	}
	return "", fmt.Errorf("the agent did not finish within 90 minutes")
}

// working reports whether the agent is currently working on the conversation.
func (p *Pipeline) working(convID string) (bool, error) {
	out, err := exec.Command("shelley", "client", "list", "-limit", "200").Output()
	if err != nil {
		return false, fmt.Errorf("shelley client list: %v", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		var c struct {
			ID      string `json:"conversation_id"`
			Working bool   `json:"working"`
		}
		if json.Unmarshal([]byte(line), &c) != nil || c.ID != convID {
			continue
		}
		return c.Working, nil
	}
	return false, fmt.Errorf("conversation %s not found", convID)
}

// lastAgentMessage returns the final message of the conversation.
func (p *Pipeline) lastAgentMessage(convID string) (string, error) {
	out, err := exec.Command("shelley", "client", "read", convID).Output()
	if err != nil {
		return "", fmt.Errorf("shelley client read: %v", err)
	}
	last := ""
	for _, line := range strings.Split(string(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var m struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if json.Unmarshal([]byte(line), &m) != nil {
			continue
		}
		if m.Type == "agent" && strings.TrimSpace(m.Text) != "" {
			last = m.Text
		}
	}
	if last == "" {
		last = "(the agent did not leave a summary)"
	}
	return last, nil
}

// --- builds ----------------------------------------------------------------

func (p *Pipeline) previewDir(id int64) string {
	return filepath.Join(p.cfg.PreviewRoot, fmt.Sprint(id))
}

// buildPreview renders the worksheets of the worktree into the preview
// directory, so the result can be looked at before it is approved.
func (p *Pipeline) buildPreview(it store.Request, worktree string) error {
	dir := p.previewDir(it.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	args := []string{"run", "./cmd/generate", "-out", dir}
	cmd := exec.Command("go", args...)
	cmd.Dir = worktree
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, tail(string(out), 800))
	}
	return nil
}

// rebuild regenerates the served output. The running server reads the generated
// manifest dynamically, so publishing worksheet content needs no binary rebuild
// or service restart.
func (p *Pipeline) rebuild() error {
	cmd := exec.Command("go", "run", "./cmd/generate", "-out", p.cfg.OutputDir)
	cmd.Dir = p.cfg.Repo
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("generate: %v: %s", err, tail(string(out), 800))
	}
	return nil
}

func tail(s string, n int) string {
	s = strings.TrimSpace(s)
	if len(s) > n {
		s = "..." + s[len(s)-n:]
	}
	return s
}

// --- decisions -------------------------------------------------------------

// Approve remains for compatibility with work completed by older server
// versions. New work is published automatically as soon as it is finished.
func (p *Pipeline) Approve(id int64) error {
	it, err := p.db.Get(id)
	if err != nil {
		return err
	}
	if it.Status != store.StatusReview {
		return fmt.Errorf("request #%d is not ready to publish", id)
	}
	return p.publishAsync(it)
}

func (p *Pipeline) publishAsync(it store.Request) error {
	lane := it.Lane()
	p.mu.Lock()
	if p.busy[lane] {
		p.mu.Unlock()
		return fmt.Errorf("request #%d is busy", it.ID)
	}
	p.busy[lane] = true
	p.mu.Unlock()

	go func() {
		defer func() {
			p.mu.Lock()
			delete(p.busy, lane)
			p.mu.Unlock()
			p.Kick()
		}()
		if err := p.publish(it, it.Note); err != nil {
			p.Logf("#%d publication failed: %v", it.ID, err)
			p.db.SetStatus(it.ID, store.StatusFailed, err.Error())
		}
	}()
	return nil
}

// publish merges a finished worksheet, rebuilds the public files and closes
// the request. The caller must hold the request lane.
func (p *Pipeline) publish(it store.Request, summary string) error {
	p.db.SetStatus(it.ID, store.StatusWorking, "publishing")
	p.publishMu.Lock()
	defer p.publishMu.Unlock()

	merged, err := p.isAncestor(it.Branch, "main")
	if err != nil {
		return err
	}
	if !merged {
		if err := p.merge(it); err != nil {
			return fmt.Errorf("merge: %w", err)
		}
	}
	p.db.SetStatus(it.ID, store.StatusWorking, "merged and pushed; rebuilding the site")
	p.Logf("#%d merged; rebuilding the site", it.ID)
	if err := p.rebuild(); err != nil {
		return fmt.Errorf("rebuild: %w", err)
	}
	if it.Kind == store.KindNew && it.Requester != "" {
		worksheets, err := site.ReadManifest(filepath.Join(p.cfg.OutputDir, site.ManifestName))
		if err != nil {
			return fmt.Errorf("record worksheet ownership: %w", err)
		}
		paths := make([]string, 0, len(worksheets))
		for _, ws := range worksheets {
			paths = append(paths, ws.Path())
		}
		if err := p.db.EnsureWorksheets(paths, it.Requester); err != nil {
			return fmt.Errorf("record worksheet ownership: %w", err)
		}
	}
	if strings.TrimSpace(summary) == "" {
		summary = "published automatically"
	}
	p.db.SetStatus(it.ID, store.StatusDone, summary)
	p.removeWorktree(it)
	p.Logf("#%d published automatically", it.ID)
	return nil
}

func (p *Pipeline) merge(it store.Request) error {
	if _, err := p.git(it.Worktree, "rebase", "main"); err != nil {
		p.git(it.Worktree, "rebase", "--abort")
		return err
	}
	if _, err := p.git(p.cfg.Repo, "merge", "--ff-only", it.Branch); err != nil {
		return err
	}
	if p.cfg.Push {
		if _, err := p.git(p.cfg.Repo, "push", "origin", "main"); err != nil {
			return err
		}
	}
	return nil
}

// Revisions returns the Git history for one worksheet, newest first.
func (p *Pipeline) Revisions(worksheet string) ([]site.Revision, error) {
	path, err := worksheetSourcePath(worksheet)
	if err != nil {
		return nil, err
	}
	out, err := p.git(p.cfg.Repo, "log", "--format=%H%x09%h%x09%ct%x09%s", "--", path)
	if err != nil {
		return nil, err
	}
	var revisions []site.Revision
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 4)
		if len(parts) != 4 {
			continue
		}
		unix, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			continue
		}
		revisions = append(revisions, site.Revision{
			Commit: parts[0], Short: parts[1], Subject: parts[3],
			Date:    time.Unix(unix, 0).Format("2 Jan 2006, 15:04"),
			Current: len(revisions) == 0,
		})
	}
	return revisions, nil
}

func worksheetSourcePath(worksheet string) (string, error) {
	parts := strings.Split(worksheet, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" ||
		parts[0] == "." || parts[0] == ".." || parts[1] == "." || parts[1] == ".." {
		return "", fmt.Errorf("invalid worksheet path %q", worksheet)
	}
	return filepath.Join("generate", parts[0], parts[1]), nil
}

// Revert restores one worksheet directory from an earlier Git revision,
// commits that restoration as a new revision, and republishes the site.
func (p *Pipeline) Revert(worksheet, commit string) error {
	path, err := worksheetSourcePath(worksheet)
	if err != nil {
		return err
	}
	revisions, err := p.Revisions(worksheet)
	if err != nil {
		return err
	}
	valid := false
	for _, revision := range revisions {
		if revision.Commit == commit && !revision.Current {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("revision is not available for %s", worksheet)
	}
	active, err := p.db.Active()
	if err != nil {
		return err
	}
	for _, it := range active {
		if it.Worksheet == worksheet {
			return fmt.Errorf("a worksheet update is already in progress")
		}
	}

	lane := "sheet:" + worksheet
	p.mu.Lock()
	if p.busy[lane] {
		p.mu.Unlock()
		return fmt.Errorf("the worksheet is busy")
	}
	p.busy[lane] = true
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		delete(p.busy, lane)
		p.mu.Unlock()
		p.Kick()
	}()

	p.publishMu.Lock()
	defer p.publishMu.Unlock()
	status, err := p.git(p.cfg.Repo, "status", "--porcelain")
	if err != nil {
		return err
	}
	if strings.TrimSpace(status) != "" {
		return fmt.Errorf("cannot revert while the main checkout has uncommitted changes")
	}
	if _, err := p.git(p.cfg.Repo, "checkout", commit, "--", path); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			p.git(p.cfg.Repo, "reset", "--hard", "HEAD")
		}
	}()
	cmd := exec.Command("git", "diff", "--cached", "--quiet", "--", path)
	cmd.Dir = p.cfg.Repo
	if err := cmd.Run(); err == nil {
		return fmt.Errorf("that revision is already current")
	} else if exit, ok := err.(*exec.ExitError); !ok || exit.ExitCode() != 1 {
		return err
	}
	message := fmt.Sprintf("Revert %s to %s", worksheet, commit[:12])
	if _, err := p.git(p.cfg.Repo, "commit", "-m", message, "--", path); err != nil {
		return err
	}
	committed = true
	if p.cfg.Push {
		if _, err := p.git(p.cfg.Repo, "push", "origin", "main"); err != nil {
			return err
		}
	}
	if err := p.rebuild(); err != nil {
		return fmt.Errorf("revision committed, but rebuild failed: %w", err)
	}
	p.Logf("%s reverted to %s and published", worksheet, commit[:12])
	return nil
}

// Reject throws the work away.
func (p *Pipeline) Reject(id int64) error {
	it, err := p.db.Get(id)
	if err != nil {
		return err
	}
	if !it.Status.Open() {
		return fmt.Errorf("request #%d is already closed", id)
	}
	p.mu.Lock()
	busy := p.busy[it.Lane()]
	p.mu.Unlock()
	if busy {
		return fmt.Errorf("request #%d is busy", id)
	}
	p.removeWorktree(it)
	p.db.SetStatus(id, store.StatusRejected, "rejected")
	p.Logf("#%d rejected", id)
	p.Kick()
	return nil
}

// Refine sends another instruction to the agent of a work item.
func (p *Pipeline) Refine(id int64, text string) error {
	it, err := p.db.Get(id)
	if err != nil {
		return err
	}
	if it.Status != store.StatusReview && it.Status != store.StatusFailed {
		return fmt.Errorf("request #%d is not waiting for a decision", id)
	}
	if err := p.db.AppendBody(id, "Refinement: "+text); err != nil {
		return err
	}
	if it.ConvID == "" {
		// Nothing to refine yet: start over with the extended request.
		p.db.SetStatus(id, store.StatusQueued, "restarted with the refinement")
		p.Kick()
		return nil
	}
	it, err = p.db.Get(id)
	if err != nil {
		return err
	}
	p.start(it, text)
	return nil
}

// Retry restarts failed agent work, or retries publication when a validated
// worksheet is already waiting in its worktree.
func (p *Pipeline) Retry(id int64) error {
	it, err := p.db.Get(id)
	if err != nil {
		return err
	}
	if it.HasPreview && it.Branch != "" && it.Worktree != "" {
		return p.publishAsync(it)
	}
	p.removeWorktree(it)
	if err := p.db.SetRun(id, "", "", ""); err != nil {
		return err
	}
	p.db.SetStatus(id, store.StatusQueued, "retrying")
	p.Kick()
	return nil
}

// Rebuild regenerates the served site on demand.
func (p *Pipeline) Rebuild() {
	go func() {
		p.publishMu.Lock()
		defer p.publishMu.Unlock()
		p.Logf("rebuilding the site")
		if err := p.rebuild(); err != nil {
			p.Logf("rebuild failed: %v", err)
			return
		}
		p.Logf("rebuild done")
	}()
}
