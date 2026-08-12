// Kommando serve liefert das output/-Verzeichnis passwortgeschuetzt aus
// (Cookie-Session, HMAC-signiert).
//
//	go run ./cmd/serve -addr :8000 -dir output
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
	"strconv"
	"strings"
	"time"
)

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

const loginPage = `<!DOCTYPE html><html lang="de"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1"><title>Lernmaterial</title>
<style>body{margin:0;height:100vh;display:flex;align-items:center;justify-content:center;
background:#fdfcfa;color:#1f3550;font:16px/1.5 system-ui,sans-serif}
form{border:2px solid #1f3550;border-radius:14px;padding:28px 30px;background:#fff;min-width:280px}
h1{margin:0 0 4px;font-size:24px}p{margin:0 0 18px;color:#6b7a8d;font-size:14px}
input{width:100%;box-sizing:border-box;padding:10px 12px;font-size:16px;border:2px solid #cfd8e3;
border-radius:8px;margin-bottom:14px}
button{width:100%;padding:10px;font-size:16px;font-weight:600;color:#fff;background:#2f9fd0;
border:0;border-radius:999px;cursor:pointer}
.err{color:#e8548c;font-size:14px;margin:-6px 0 12px}</style></head><body>
<form method="POST"><h1>Lernmaterial</h1><p>Bitte Passwort eingeben.</p>
{{ERR}}<input type="password" name="pw" autofocus autocomplete="current-password" placeholder="Passwort">
<button type="submit">Weiter</button></form></body></html>`

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
			errMsg = `<div class="err">Falsches Passwort.</div>`
		}
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusUnauthorized)
		io.WriteString(w, strings.Replace(loginPage, "{{ERR}}", errMsg, 1))
	})
}

func subtleEq(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

func main() {
	addr := flag.String("addr", ":8000", "listen address")
	dir := flag.String("dir", "output", "directory to serve")
	flag.Parse()

	secret = []byte(os.Getenv("SITE_SECRET"))
	if len(secret) == 0 {
		secret = make([]byte, 32)
		if _, err := rand.Read(secret); err != nil {
			log.Fatal(err)
		}
	}
	log.Printf("serving %s on %s", *dir, *addr)
	log.Fatal(http.ListenAndServe(*addr, auth(http.FileServer(http.Dir(*dir)))))
}
