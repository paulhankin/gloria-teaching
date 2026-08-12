// Command serve exposes the output/ directory behind a password
// (cookie session, HMAC signed) and renders the interactive front page
// (worksheet index + request forms).
//
//	go run ./cmd/serve -addr :8000 -dir output -db data/requests.db
package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"learningmaterial/internal/sheet"
	"learningmaterial/internal/site"
	"learningmaterial/internal/store"

	_ "learningmaterial/generate/math/number_sequences"
	_ "learningmaterial/generate/math/price_puzzles"
	_ "learningmaterial/generate/math/venn_diagrams"
)

var db *store.DB

const cookieName = "lm_auth"

var (
	password = envOr("SITE_PASSWORD", "sixseven")
	secret   []byte
)

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// token = exp|hmac(exp)
func makeToken(exp int64) string {
	m := hmac.New(sha256.New, secret)
	fmt.Fprintf(m, "%d", exp)
	return fmt.Sprintf("%d|%s", exp, hex.EncodeToString(m.Sum(nil)))
}

func validToken(t string) bool {
	parts := strings.SplitN(t, "|", 2)
	if len(parts) != 2 {
		return false
	}
	exp, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return hmac.Equal([]byte(makeToken(exp)), []byte(t))
}

const loginPage = `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>Learning material</title>
<style>body{margin:0;height:100vh;display:flex;align-items:center;justify-content:center;
background:#fdfcfa;color:#1f3550;font:16px/1.5 system-ui,sans-serif}
form{border:2px solid #1f3550;border-radius:14px;padding:28px 30px;background:#fff;min-width:280px}
h1{margin:0 0 4px;font-size:24px}p{margin:0 0 18px;color:#6b7a8d;font-size:14px}
input{width:100%;box-sizing:border-box;padding:10px 12px;font-size:16px;border:2px solid #cfd8e3;
border-radius:8px;margin-bottom:14px}
button{width:100%;padding:10px;font-size:16px;font-weight:600;color:#fff;background:#2f9fd0;
border:0;border-radius:999px;cursor:pointer}
.err{color:#e8548c;font-size:14px;margin:-6px 0 12px}</style></head><body>
<form method="POST"><h1>Learning material</h1><p>Please enter the password.</p>
{{ERR}}<input type="password" name="pw" autofocus autocomplete="current-password" placeholder="Password">
<button type="submit">Continue</button></form></body></html>`

func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie(cookieName); err == nil && validToken(c.Value) {
			next.ServeHTTP(w, r)
			return
		}
		errMsg := ""
		if r.Method == http.MethodPost {
			r.ParseForm()
			if subtleEq(r.FormValue("pw"), password) {
				exp := time.Now().Add(365 * 24 * time.Hour).Unix()
				http.SetCookie(w, &http.Cookie{
					Name: cookieName, Value: makeToken(exp), Path: "/",
					Expires: time.Unix(exp, 0), HttpOnly: true, Secure: true,
					SameSite: http.SameSiteLaxMode,
				})
				http.Redirect(w, r, r.URL.Path, http.StatusSeeOther)
				return
			}
			errMsg = `<div class="err">Wrong password.</div>`
		}
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, strings.Replace(loginPage, "{{ERR}}", errMsg, 1))
	})
}

func subtleEq(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

// index renders the front page with the request UI.
func index(w http.ResponseWriter, r *http.Request) {
	d := site.Data{Worksheets: sheet.All(), Admin: db.AdminMode(), Flash: flash(w, r)}
	if d.Admin {
		rs, err := db.All()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		d.Requests = rs
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
	} else if kind != store.KindChange || !knownWorksheet(worksheet) {
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
	setFlash(w, string(kind))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// deleteRequest removes a request (admin mode only).
func deleteRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	if !db.AdminMode() {
		http.Error(w, "admin mode is off", http.StatusForbidden)
		return
	}
	r.ParseForm()
	id, err := strconv.ParseInt(r.FormValue("id"), 10, 64)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	if err := db.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setFlash(w, "deleted")
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

func knownWorksheet(path string) bool {
	for _, ws := range sheet.All() {
		if ws.Path() == path {
			return true
		}
	}
	return false
}

func main() {
	addr := flag.String("addr", ":8000", "listen address")
	dir := flag.String("dir", "output", "directory to serve")
	dbPath := flag.String("db", "data/requests.db", "SQLite database with the requests")
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

	secret = []byte(os.Getenv("SITE_SECRET"))
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			log.Fatal(err)
		}
	}
	files := http.FileServer(http.Dir(*dir))
	mux := http.NewServeMux()
	mux.HandleFunc("/requests", postRequest)
	mux.HandleFunc("/requests/delete", deleteRequest)
	mux.HandleFunc("/admin", postAdmin)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			index(w, r)
			return
		}
		files.ServeHTTP(w, r)
	})

	log.Printf("serving %s on %s (db %s)", *dir, *addr, *dbPath)
	log.Fatal(http.ListenAndServe(*addr, auth(mux)))
}
