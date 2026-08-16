// Command serve exposes the output/ directory behind username/email/password accounts
// and renders the interactive front page (worksheet index + request forms).
//
//	go run ./cmd/serve -addr :8000 -dir output -db data/requests.db
package main

import (
	"crypto/rand"
	"flag"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"learningmaterial/internal/account"
	"learningmaterial/internal/pipeline"
	"learningmaterial/internal/site"
	"learningmaterial/internal/store"
)

var db *store.DB
var pipe *pipeline.Pipeline
var manifestPath string
var outputRoot string

const existingWorksheetOwner = "g.n.hankin@gmail.com"

func siteBaseURL() string {
	if value := strings.TrimRight(os.Getenv("SITE_BASE_URL"), "/"); value != "" {
		return value
	}
	return "https://gloria-teaching.exe.xyz"
}

func adminEmails() []string {
	return []string{"paul.hankin@pobox.com", "g.n.hankin@gmail.com"}
}

func allowedEmails() []string {
	value := os.Getenv("SITE_ALLOWED_EMAILS")
	if value == "" {
		value = "paul.hankin@pobox.com,g.n.hankin@gmail.com"
	}
	seen := make(map[string]bool)
	var emails []string
	for _, email := range append(strings.Split(value, ","), adminEmails()...) {
		if email = strings.ToLower(strings.TrimSpace(email)); email != "" && !seen[email] {
			seen[email] = true
			emails = append(emails, email)
		}
	}
	return emails
}

// index renders the front page with the request UI.
func index(w http.ResponseWriter, r *http.Request) {
	manifest, err := site.ReadManifest(manifestPath)
	if err != nil {
		http.Error(w, "worksheet catalog: "+err.Error(), http.StatusInternalServerError)
		return
	}
	worksheets, err := ownedWorksheets(manifest, account.Email(r))
	if err != nil {
		http.Error(w, "worksheet access: "+err.Error(), http.StatusInternalServerError)
		return
	}
	revisions := make(map[string][]site.Revision, len(worksheets))
	for _, ws := range worksheets {
		history, err := pipe.Revisions(ws.Path())
		if err != nil {
			http.Error(w, "worksheet revisions: "+err.Error(), http.StatusInternalServerError)
			return
		}
		revisions[ws.Path()] = history
	}
	d := site.Data{
		Worksheets:     worksheets,
		Revisions:      revisions,
		Admin:          db.AdminMode(),
		Flash:          flash(w, r),
		User:           account.Username(r),
		Actor:          account.ActorUsername(r),
		CanImpersonate: account.IsAdmin(r),
	}
	if d.CanImpersonate {
		users, err := db.Users()
		if err != nil {
			http.Error(w, "users: "+err.Error(), http.StatusInternalServerError)
			return
		}
		for _, user := range users {
			d.Users = append(d.Users, user.Username)
		}
	}
	rs, err := db.All()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	d.Requests = rs
	if d.Admin {
		d.Log = pipe.Log()
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	io.WriteString(w, site.Index(d))
}

// flash reads and clears the one-shot confirmation message.
func flash(w http.ResponseWriter, r *http.Request) string {
	c, err := r.Cookie("lm_flash")
	if err != nil || c.Value == "" {
		return ""
	}
	http.SetCookie(w, &http.Cookie{Name: "lm_flash", Value: "", Path: "/", MaxAge: -1})
	switch c.Value {
	case "new":
		return "Thanks! Your worksheet request has been saved."
	case "change":
		return "Thanks! Your change request has been saved."
	case "deleted":
		return "Request deleted."
	case "approve":
		return "Approved: merging, pushing and rebuilding."
	case "reject":
		return "Rejected: the change was thrown away."
	case "refine":
		return "Refinement sent to the agent."
	case "retry":
		return "Restarted."
	case "rebuild":
		return "Rebuilding the site."
	case "sharing":
		return "Sharing settings saved."
	case "reverted":
		return "Earlier revision restored and published."
	}
	return ""
}

func setFlash(w http.ResponseWriter, v string) {
	http.SetCookie(w, &http.Cookie{Name: "lm_flash", Value: v, Path: "/", MaxAge: 300})
}

// postRequest stores a new or change request.
func postRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	kind := store.Kind(r.FormValue("kind"))
	worksheet := r.FormValue("worksheet")
	if kind == store.KindNew {
		worksheet = ""
	} else if kind != store.KindChange {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	} else if known, err := knownOwnedWorksheet(worksheet, account.Email(r)); err != nil {
		http.Error(w, "worksheet catalog: "+err.Error(), http.StatusInternalServerError)
		return
	} else if !known {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	body := strings.TrimSpace(r.FormValue("body"))
	if body == "" {
		http.Error(w, "empty request", http.StatusBadRequest)
		return
	}
	req := store.Request{
		Kind:      kind,
		Worksheet: worksheet,
		Author:    account.Username(r),
		Requester: account.Email(r),
		Body:      body,
	}
	if _, err := db.Add(req); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pipe.Kick()
	setFlash(w, string(kind))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// deleteRequest removes a request (admin mode only).
func deleteRequest(w http.ResponseWriter, r *http.Request) {
	id, ok := adminPost(w, r)
	if !ok {
		return
	}
	if err := db.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setFlash(w, "deleted")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// adminPost validates a POST with an id field in admin mode.
func adminPost(w http.ResponseWriter, r *http.Request) (int64, bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return 0, false
	}
	if !db.AdminMode() {
		http.Error(w, "admin mode is off", http.StatusForbidden)
		return 0, false
	}
	r.ParseForm()
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

// work handles the decisions on a work item: approve, reject, refine, retry.
func work(w http.ResponseWriter, r *http.Request) {
	action := strings.TrimPrefix(r.URL.Path, "/work/")
	if action == "rebuild" {
		if r.Method != http.MethodPost || !db.AdminMode() {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		pipe.Rebuild()
		setFlash(w, "rebuild")
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	id, ok := adminPost(w, r)
	if !ok {
		return
	}
	var err error
	switch action {
	case "approve":
		err = pipe.Approve(id)
	case "reject":
		err = pipe.Reject(id)
	case "retry":
		err = pipe.Retry(id)
	case "refine":
		body := strings.TrimSpace(r.FormValue("body"))
		if body == "" {
			http.Error(w, "empty refinement", http.StatusBadRequest)
			return
		}
		err = pipe.Refine(id, body)
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	setFlash(w, action)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// postAdmin toggles admin mode.
func postAdmin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	r.ParseForm()
	if err := db.SetAdminMode(r.FormValue("admin") == "on"); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func knownOwnedWorksheet(path, owner string) (bool, error) {
	worksheets, err := site.ReadManifest(manifestPath)
	if err != nil {
		return false, err
	}
	known := false
	for _, ws := range worksheets {
		if ws.Path() == path {
			known = true
			break
		}
	}
	if !known {
		return false, nil
	}
	access, err := db.WorksheetByPath(path)
	if err != nil {
		return false, err
	}
	return strings.EqualFold(access.OwnerEmail, owner), nil
}

func ownedWorksheets(manifest []site.Worksheet, owner string) ([]site.Worksheet, error) {
	paths := make([]string, 0, len(manifest))
	byPath := make(map[string]site.Worksheet, len(manifest))
	for _, ws := range manifest {
		paths = append(paths, ws.Path())
		byPath[ws.Path()] = ws
	}
	if err := db.EnsureWorksheets(paths, existingWorksheetOwner); err != nil {
		return nil, err
	}
	access, err := db.WorksheetsOwnedBy(owner)
	if err != nil {
		return nil, err
	}
	out := make([]site.Worksheet, 0, len(access))
	for _, a := range access {
		ws, ok := byPath[a.Worksheet]
		if !ok {
			continue
		}
		ws.Owner = a.OwnerEmail
		ws.Visibility = a.Visibility
		ws.Shares = a.Shares
		out = append(out, ws)
	}
	return out, nil
}

func worksheetRevert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	worksheet := r.FormValue("worksheet")
	known, err := knownOwnedWorksheet(worksheet, account.Email(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !known {
		http.Error(w, "worksheet not found", http.StatusNotFound)
		return
	}
	if err := pipe.Revert(worksheet, r.FormValue("commit")); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	setFlash(w, "reverted")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func worksheetVisibility(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	visibility := store.Visibility(r.FormValue("visibility"))
	if err := db.SetWorksheetVisibility(r.FormValue("worksheet"), account.Email(r), visibility); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setFlash(w, "sharing")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func worksheetShare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	user, err := db.UserByEmail(email)
	if err != nil || !user.Verified {
		http.Error(w, "share recipient must be an existing verified user", http.StatusBadRequest)
		return
	}
	if err := db.SetWorksheetShare(
		r.FormValue("worksheet"), account.Email(r), user.Email, store.Permission(r.FormValue("permission")),
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setFlash(w, "sharing")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func worksheetShareDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	if err := db.DeleteWorksheetShare(
		r.FormValue("worksheet"), account.Email(r), strings.TrimSpace(r.FormValue("email")),
	); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	setFlash(w, "sharing")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func publicWorksheets(manifest []site.Worksheet, username, ownerEmail string) ([]site.Worksheet, error) {
	owned, err := ownedWorksheets(manifest, ownerEmail)
	if err != nil {
		return nil, err
	}
	out := make([]site.Worksheet, 0, len(owned))
	for _, ws := range owned {
		if ws.Username == username && ws.Visibility == store.VisibilityPublic {
			out = append(out, ws)
		}
	}
	return out, nil
}

func worksheetForUser(manifest []site.Worksheet, username, name string) (site.Worksheet, bool) {
	for _, ws := range manifest {
		if ws.Username == username && ws.Name == name {
			return ws, true
		}
	}
	return site.Worksheet{}, false
}

func worksheetRoutes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "GET only", http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) == 3 && parts[0] == "worksheets" && parts[2] == "index" {
		publicWorksheetIndex(w, r, parts[1])
		return
	}
	if len(parts) == 4 && parts[0] == "worksheets" && parts[2] == "sheet" {
		serveWorksheet(w, r, parts[1], parts[3])
		return
	}
	http.NotFound(w, r)
}

func publicWorksheetIndex(w http.ResponseWriter, r *http.Request, username string) {
	owner, err := db.UserByUsername(username)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	manifest, err := site.ReadManifest(manifestPath)
	if err != nil {
		http.Error(w, "worksheet catalog: "+err.Error(), http.StatusInternalServerError)
		return
	}
	worksheets, err := publicWorksheets(manifest, owner.Username, owner.Email)
	if err != nil {
		http.Error(w, "worksheet access: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	io.WriteString(w, site.PublicIndex(site.PublicData{
		OwnerUsername: owner.Username, ViewerUsername: account.Username(r), Worksheets: worksheets,
	}))
}

func serveWorksheet(w http.ResponseWriter, r *http.Request, username, name string) {
	owner, err := db.UserByUsername(username)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	manifest, err := site.ReadManifest(manifestPath)
	if err != nil {
		http.Error(w, "worksheet catalog: "+err.Error(), http.StatusInternalServerError)
		return
	}
	ws, ok := worksheetForUser(manifest, owner.Username, name)
	if !ok {
		http.NotFound(w, r)
		return
	}
	access, err := db.WorksheetByPath(ws.Path())
	if err != nil || !strings.EqualFold(access.OwnerEmail, owner.Email) {
		http.NotFound(w, r)
		return
	}
	allowed, err := db.CanViewWorksheet(ws.Path(), account.Email(r))
	if err != nil {
		http.Error(w, "worksheet access: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if !allowed {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, filepath.Join(outputRoot, ws.OutputPath(), "index.html"))
}

func canServeWorksheet(path, viewerEmail string) bool {
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 3 {
		return true
	}
	manifest, err := site.ReadManifest(manifestPath)
	if err != nil {
		return false
	}
	outputPath := parts[0] + "/" + parts[1]
	for _, ws := range manifest {
		if ws.OutputPath() != outputPath {
			continue
		}
		allowed, err := db.CanViewWorksheet(ws.Path(), viewerEmail)
		return err == nil && allowed
	}
	return false
}

func main() {
	addr := flag.String("addr", ":8000", "listen address")
	dir := flag.String("dir", "output", "directory to serve")
	dbPath := flag.String("db", "data/requests.db", "SQLite database with the requests")
	repo := flag.String("repo", ".", "git checkout the pipeline works on")
	workRoot := flag.String("work", "data/worktrees", "directory for the per-item git worktrees")
	usersRoot := flag.String("users", "/users", "directory containing local per-user worksheet repositories")
	preview := flag.String("preview", "data/preview", "directory for the per-item preview builds")
	sandbox := flag.Bool("sandbox", false, "run agent work in per-request sandboxes with standalone repository clones")
	sandboxRoot := flag.String("sandboxes", "data/sandboxes", "parent directory for the per-request sandboxes")
	push := flag.Bool("push", true, "push to origin after an approved change")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		log.Fatal(err)
	}
	var err error
	db, err = store.Open(*dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	secret := []byte(os.Getenv("SITE_SECRET"))
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			log.Fatal(err)
		}
	}
	abs := func(p string) string {
		a, err := filepath.Abs(p)
		if err != nil {
			log.Fatal(err)
		}
		return a
	}
	outputDir := abs(*dir)
	outputRoot = outputDir
	manifestPath = filepath.Join(outputDir, site.ManifestName)
	manifest, err := site.ReadManifest(manifestPath)
	if err != nil {
		log.Fatalf("worksheet catalog: %v", err)
	}
	paths := make([]string, 0, len(manifest))
	for _, ws := range manifest {
		paths = append(paths, ws.Path())
	}
	if err := db.EnsureWorksheets(paths, existingWorksheetOwner); err != nil {
		log.Fatalf("worksheet access: %v", err)
	}
	pipe = pipeline.New(db, pipeline.Config{
		Repo:          abs(*repo),
		WorksheetRoot: abs(*usersRoot),
		WorkRoot:      abs(*workRoot),
		PreviewRoot:   abs(*preview),
		OutputDir:     outputDir,
		Push:          *push,
		Sandbox:       *sandbox,
		SandboxRoot:   abs(*sandboxRoot),
	})
	pipe.Start()

	// SIGTERM (systemd stop) and SIGINT (Ctrl-C) gracefully stop every
	// sandboxed Shelley server before the process exits, so no sandboxed
	// agent outlives the service.
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigs
		log.Printf("received %s; stopping sandboxed shelley servers", sig)
		pipe.Shutdown()
		os.Exit(0)
	}()

	files := http.FileServer(http.Dir(*dir))
	previews := http.StripPrefix("/preview/", http.FileServer(http.Dir(abs(*preview))))
	mux := http.NewServeMux()
	mux.HandleFunc("/requests", postRequest)
	mux.HandleFunc("/requests/delete", deleteRequest)
	mux.HandleFunc("/worksheets/visibility", worksheetVisibility)
	mux.HandleFunc("/worksheets/revert", worksheetRevert)
	mux.HandleFunc("/worksheets/shares", worksheetShare)
	mux.HandleFunc("/worksheets/shares/delete", worksheetShareDelete)
	mux.HandleFunc("/worksheets/", worksheetRoutes)
	mux.HandleFunc("/work/", work)
	mux.HandleFunc("/admin", postAdmin)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/" || r.URL.Path == "/index.html":
			index(w, r)
		case strings.HasPrefix(r.URL.Path, "/preview/"):
			if !db.AdminMode() {
				http.Error(w, "admin mode is off", http.StatusForbidden)
				return
			}
			w.Header().Set("Cache-Control", "no-store")
			previews.ServeHTTP(w, r)
		default:
			if !canServeWorksheet(r.URL.Path, account.Email(r)) {
				http.NotFound(w, r)
				return
			}
			files.ServeHTTP(w, r)
		}
	})

	accounts := account.New(db, secret, allowedEmails(), adminEmails(), account.GatewayMailer{}, siteBaseURL())
	public := http.NewServeMux()
	accounts.Register(public)
	public.Handle("/", accounts.RequireAccess(mux))

	log.Printf("serving %s on %s (db %s)", *dir, *addr, *dbPath)
	log.Fatal(http.ListenAndServe(*addr, public))
}
