// Command serve exposes the output/ directory behind email/password accounts
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
	"path/filepath"
	"strconv"
	"strings"

	"learningmaterial/internal/account"
	"learningmaterial/internal/pipeline"
	"learningmaterial/internal/site"
	"learningmaterial/internal/store"
)

var db *store.DB
var pipe *pipeline.Pipeline
var manifestPath string

func siteBaseURL() string {
	if value := strings.TrimRight(os.Getenv("SITE_BASE_URL"), "/"); value != "" {
		return value
	}
	return "https://gloria-teaching.exe.xyz"
}

func allowedEmails() []string {
	value := os.Getenv("SITE_ALLOWED_EMAILS")
	if value == "" {
		value = "paul.hankin@pobox.com,g.n.hankin@gmail.com"
	}
	var emails []string
	for _, email := range strings.Split(value, ",") {
		if email = strings.TrimSpace(email); email != "" {
			emails = append(emails, email)
		}
	}
	return emails
}

// index renders the front page with the request UI.
func index(w http.ResponseWriter, r *http.Request) {
	worksheets, err := site.ReadManifest(manifestPath)
	if err != nil {
		http.Error(w, "worksheet catalog: "+err.Error(), http.StatusInternalServerError)
		return
	}
	d := site.Data{Worksheets: worksheets, Admin: db.AdminMode(), Flash: flash(w, r), User: account.Email(r)}
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
	} else if known, err := knownWorksheet(worksheet); err != nil {
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
		Author:    strings.TrimSpace(r.FormValue("author")),
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

func knownWorksheet(path string) (bool, error) {
	worksheets, err := site.ReadManifest(manifestPath)
	if err != nil {
		return false, err
	}
	for _, ws := range worksheets {
		if ws.Path() == path {
			return true, nil
		}
	}
	return false, nil
}

func main() {
	addr := flag.String("addr", ":8000", "listen address")
	dir := flag.String("dir", "output", "directory to serve")
	dbPath := flag.String("db", "data/requests.db", "SQLite database with the requests")
	repo := flag.String("repo", ".", "git checkout the pipeline works on")
	workRoot := flag.String("work", "data/worktrees", "directory for the per-item git worktrees")
	preview := flag.String("preview", "data/preview", "directory for the per-item preview builds")
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
	manifestPath = filepath.Join(outputDir, site.ManifestName)
	if _, err := site.ReadManifest(manifestPath); err != nil {
		log.Fatalf("worksheet catalog: %v", err)
	}
	pipe = pipeline.New(db, pipeline.Config{
		Repo:        abs(*repo),
		WorkRoot:    abs(*workRoot),
		PreviewRoot: abs(*preview),
		OutputDir:   outputDir,
		Push:        *push,
	})
	pipe.Start()

	files := http.FileServer(http.Dir(*dir))
	previews := http.StripPrefix("/preview/", http.FileServer(http.Dir(abs(*preview))))
	mux := http.NewServeMux()
	mux.HandleFunc("/requests", postRequest)
	mux.HandleFunc("/requests/delete", deleteRequest)
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
			files.ServeHTTP(w, r)
		}
	})

	accounts := account.New(db, secret, allowedEmails(), account.GatewayMailer{}, siteBaseURL())
	public := http.NewServeMux()
	accounts.Register(public)
	public.Handle("/", accounts.RequireAccess(mux))

	log.Printf("serving %s on %s (db %s)", *dir, *addr, *dbPath)
	log.Fatal(http.ListenAndServe(*addr, public))
}
