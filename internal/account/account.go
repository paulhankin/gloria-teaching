// Package account provides email/password accounts for the website.
package account

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"learningmaterial/internal/store"
)

const (
	cookieName    = "lm_account"
	verifyPurpose = "verify-email"
	resetPurpose  = "reset-password"
)

type contextKey string

const emailContextKey contextKey = "account-email"
const actorEmailContextKey contextKey = "account-actor-email"
const adminContextKey contextKey = "account-admin"

// Mailer sends account emails.
type Mailer interface {
	Send(to, subject, body string) error
}

// GatewayMailer sends mail through the exe.dev email gateway.
type GatewayMailer struct {
	URL    string
	Client *http.Client
}

func (m GatewayMailer) Send(to, subject, body string) error {
	gateway := m.URL
	if gateway == "" {
		gateway = "http://169.254.169.254/gateway/email/send"
	}
	client := m.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	payload, err := json.Marshal(map[string]string{"to": to, "subject": subject, "body": body})
	if err != nil {
		return err
	}
	resp, err := client.Post(gateway, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("email gateway returned %s", resp.Status)
	}
	return nil
}

// Manager owns account routes and session validation.
type Manager struct {
	db      *store.DB
	secret  []byte
	allowed map[string]bool
	admins  map[string]bool
	mailer  Mailer
	baseURL string
}

func New(db *store.DB, secret []byte, allowed, admins []string, mailer Mailer, baseURL string) *Manager {
	a := make(map[string]bool, len(allowed))
	for _, email := range allowed {
		a[normalizeEmail(email)] = true
	}
	adminSet := make(map[string]bool, len(admins))
	for _, email := range admins {
		adminSet[normalizeEmail(email)] = true
	}
	return &Manager{db: db, secret: secret, allowed: a, admins: adminSet, mailer: mailer, baseURL: strings.TrimRight(baseURL, "/")}
}

// Register installs public account-management routes.
func (m *Manager) Register(mux *http.ServeMux) {
	mux.HandleFunc("/account/create", m.create)
	mux.HandleFunc("/account/sign-in", m.signIn)
	mux.HandleFunc("/account/sign-out", m.signOut)
	mux.HandleFunc("/account/impersonate", m.impersonate)
	mux.HandleFunc("/account/verify", m.verify)
	mux.HandleFunc("/account/forgot-password", m.forgotPassword)
	mux.HandleFunc("/account/reset-password", m.resetPassword)
}

// RequireAccess permits only signed-in accounts on the allowlist.
func (m *Manager) RequireAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actor, email, ok := m.sessionIdentity(r)
		if !ok || !m.allowed[actor] {
			nextURL := r.URL.RequestURI()
			http.Redirect(w, r, "/account/sign-in?next="+url.QueryEscape(nextURL), http.StatusSeeOther)
			return
		}
		ctx := context.WithValue(r.Context(), emailContextKey, email)
		ctx = context.WithValue(ctx, actorEmailContextKey, actor)
		ctx = context.WithValue(ctx, adminContextKey, m.admins[actor])
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Email returns the effective account email from a protected request.
func Email(r *http.Request) string {
	email, _ := r.Context().Value(emailContextKey).(string)
	return email
}

// ActorEmail returns the account that originally signed in.
func ActorEmail(r *http.Request) string {
	email, _ := r.Context().Value(actorEmailContextKey).(string)
	return email
}

// IsAdmin reports whether the signed-in account may impersonate users.
func IsAdmin(r *http.Request) bool {
	admin, _ := r.Context().Value(adminContextKey).(bool)
	return admin
}

func (m *Manager) create(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		m.render(w, pageData{Page: "create"})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	email := normalizeEmail(r.FormValue("email"))
	password := r.FormValue("password")
	if !validEmail(email) {
		m.render(w, pageData{Page: "create", Email: email, Error: "Enter a valid email address."})
		return
	}
	if err := validPassword(password); err != nil {
		m.render(w, pageData{Page: "create", Email: email, Error: err.Error()})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "could not create account", http.StatusInternalServerError)
		return
	}
	user, err := m.db.CreateUser(email, hash)
	if err != nil {
		if _, lookupErr := m.db.UserByEmail(email); lookupErr == nil {
			m.render(w, pageData{Page: "create", Email: email, Error: "An account with that email address already exists."})
			return
		}
		http.Error(w, "could not create account", http.StatusInternalServerError)
		return
	}
	if !m.allowed[email] {
		m.render(w, pageData{Page: "message", Title: "Account created", Message: "Your account has been created, but access to the site has not been granted yet."})
		return
	}
	if err := m.sendToken(r, user, verifyPurpose); err != nil {
		log.Printf("account: sending verification email to %s: %v", email, err)
		m.render(w, pageData{Page: "message", Title: "Account created", Message: "Your account was created, but the confirmation email could not be sent. Sign in to try sending it again."})
		return
	}
	m.render(w, pageData{Page: "message", Title: "Check your email", Message: "Use the confirmation link we sent before signing in."})
}

func (m *Manager) signIn(w http.ResponseWriter, r *http.Request) {
	next := safeNext(r.URL.Query().Get("next"))
	if r.Method == http.MethodGet {
		m.render(w, pageData{Page: "sign-in", Next: next})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	next = safeNext(r.FormValue("next"))
	email := normalizeEmail(r.FormValue("email"))
	user, err := m.db.UserByEmail(email)
	if err != nil || bcrypt.CompareHashAndPassword(user.PasswordHash, []byte(r.FormValue("password"))) != nil {
		m.render(w, pageData{Page: "sign-in", Email: email, Next: next, Error: "Email address or password is incorrect."})
		return
	}
	if !m.allowed[email] {
		m.render(w, pageData{Page: "message", Title: "Access not granted", Message: "Your account exists, but it is not currently allowed to access this site."})
		return
	}
	if !user.Verified {
		message := "Confirm your email address before signing in. We sent you a new confirmation link."
		if err := m.sendToken(r, user, verifyPurpose); err != nil {
			log.Printf("account: resending verification email to %s: %v", email, err)
			message = "Confirm your email address before signing in. We could not send a new confirmation email just now."
		}
		m.render(w, pageData{Page: "message", Title: "Check your email", Message: message})
		return
	}
	m.setSession(w, email, email)
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (m *Manager) signOut(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/account/sign-in", http.StatusSeeOther)
}

func (m *Manager) impersonate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST only", http.StatusMethodNotAllowed)
		return
	}
	actor, _, ok := m.sessionIdentity(r)
	if !ok || !m.allowed[actor] || !m.admins[actor] {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	target := normalizeEmail(r.FormValue("email"))
	if target == "" {
		target = actor
	}
	user, err := m.db.UserByEmail(target)
	if err != nil {
		http.Error(w, "user not found", http.StatusBadRequest)
		return
	}
	m.setSession(w, actor, normalizeEmail(user.Email))
	http.Redirect(w, r, safeNext(r.FormValue("next")), http.StatusSeeOther)
}

func (m *Manager) verify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token, err := decodeToken(r.URL.Query().Get("token"))
	if err != nil {
		m.render(w, pageData{Page: "message", Title: "Invalid link", Message: "This confirmation link is invalid or has expired."})
		return
	}
	used, err := m.db.ConsumeAuthToken(tokenHash(token), verifyPurpose)
	if err != nil {
		m.render(w, pageData{Page: "message", Title: "Invalid link", Message: "This confirmation link is invalid or has expired."})
		return
	}
	if err := m.db.MarkUserVerified(used.UserID); err != nil {
		http.Error(w, "could not verify account", http.StatusInternalServerError)
		return
	}
	m.render(w, pageData{Page: "message", Title: "Email confirmed", Message: "Your account is ready. You can now sign in."})
}

func (m *Manager) forgotPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		m.render(w, pageData{Page: "forgot"})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	email := normalizeEmail(r.FormValue("email"))
	if user, err := m.db.UserByEmail(email); err == nil {
		if err := m.sendToken(r, user, resetPurpose); err != nil {
			log.Printf("account: sending password reset email to %s: %v", email, err)
		}
	}
	m.render(w, pageData{Page: "message", Title: "Check your email", Message: "If an account exists for that address, we sent a password reset link."})
}

func (m *Manager) resetPassword(w http.ResponseWriter, r *http.Request) {
	tokenText := r.URL.Query().Get("token")
	if r.Method == http.MethodGet {
		if _, err := decodeToken(tokenText); err != nil {
			m.render(w, pageData{Page: "message", Title: "Invalid link", Message: "This password reset link is invalid or has expired."})
			return
		}
		m.render(w, pageData{Page: "reset", Token: tokenText})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "bad form", http.StatusBadRequest)
		return
	}
	tokenText = r.FormValue("token")
	password := r.FormValue("password")
	if err := validPassword(password); err != nil {
		m.render(w, pageData{Page: "reset", Token: tokenText, Error: err.Error()})
		return
	}
	token, err := decodeToken(tokenText)
	if err != nil {
		m.render(w, pageData{Page: "message", Title: "Invalid link", Message: "This password reset link is invalid or has expired."})
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, "could not reset password", http.StatusInternalServerError)
		return
	}
	used, err := m.db.ConsumeAuthToken(tokenHash(token), resetPurpose)
	if err != nil {
		m.render(w, pageData{Page: "message", Title: "Invalid link", Message: "This password reset link is invalid or has expired."})
		return
	}
	if err := m.db.SetUserPassword(used.UserID, hash); err != nil {
		http.Error(w, "could not reset password", http.StatusInternalServerError)
		return
	}
	m.render(w, pageData{Page: "message", Title: "Password updated", Message: "You can now sign in with your new password."})
}

func (m *Manager) sendToken(r *http.Request, user store.User, purpose string) error {
	if m.mailer == nil {
		return errors.New("no mailer configured")
	}
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return err
	}
	expires := time.Now().Add(24 * time.Hour)
	path := "/account/verify"
	subject := "Confirm your Learning material account"
	bodyIntro := "Confirm your email address by opening this link:"
	if purpose == resetPurpose {
		expires = time.Now().Add(time.Hour)
		path = "/account/reset-password"
		subject = "Reset your Learning material password"
		bodyIntro = "Reset your password by opening this link:"
	}
	if err := m.db.PutAuthToken(user.ID, purpose, tokenHash(token), expires); err != nil {
		return err
	}
	baseURL := m.baseURL
	if baseURL == "" {
		baseURL = requestBaseURL(r)
	}
	link := baseURL + path + "?token=" + url.QueryEscape(base64.RawURLEncoding.EncodeToString(token))
	body := bodyIntro + "\n\n" + link + "\n\nThis link expires " + expires.Format("2 Jan 2006 at 15:04 MST") + "."
	return m.mailer.Send(user.Email, subject, body)
}

func tokenHash(token []byte) []byte {
	h := sha256.Sum256(token)
	return h[:]
}

func decodeToken(s string) ([]byte, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil || len(b) != 32 {
		return nil, errors.New("invalid token")
	}
	return b, nil
}

func (m *Manager) setSession(w http.ResponseWriter, actor, email string) {
	exp := time.Now().Add(30 * 24 * time.Hour).Unix()
	payload := normalizeEmail(actor) + "\n" + normalizeEmail(email) + "\n" + strconv.FormatInt(exp, 10)
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(payload))
	value := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + hex.EncodeToString(mac.Sum(nil))
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: value, Path: "/", Expires: time.Unix(exp, 0), MaxAge: 30 * 24 * 60 * 60, HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode})
}

func (m *Manager) sessionIdentity(r *http.Request) (actor, email string, ok bool) {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return "", "", false
	}
	parts := strings.SplitN(cookie.Value, ".", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", false
	}
	mac := hmac.New(sha256.New, m.secret)
	mac.Write(payload)
	want, err := hex.DecodeString(parts[1])
	if err != nil || !hmac.Equal(mac.Sum(nil), want) {
		return "", "", false
	}
	fields := strings.Split(string(payload), "\n")
	var expText string
	switch len(fields) {
	case 2: // Sessions issued before impersonation support.
		actor, email, expText = fields[0], fields[0], fields[1]
	case 3:
		actor, email, expText = fields[0], fields[1], fields[2]
	default:
		return "", "", false
	}
	exp, err := strconv.ParseInt(expText, 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return "", "", false
	}
	actor, email = normalizeEmail(actor), normalizeEmail(email)
	if actor == "" || email == "" {
		return "", "", false
	}
	return actor, email, true
}

func normalizeEmail(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func validEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s && strings.Contains(s, "@") && len(s) <= 254
}

func validPassword(password string) error {
	if len(password) < 10 {
		return errors.New("Use at least 10 characters for your password.")
	}
	if len(password) > 72 {
		return errors.New("Use no more than 72 characters for your password.")
	}
	return nil
}

func safeNext(s string) string {
	if s == "" || !strings.HasPrefix(s, "/") || strings.HasPrefix(s, "//") {
		return "/"
	}
	return s
}

func requestBaseURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return scheme + "://" + host
}

type pageData struct {
	Page    string
	Title   string
	Message string
	Error   string
	Email   string
	Next    string
	Token   string
}

func (m *Manager) render(w http.ResponseWriter, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := pageTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

var pageTemplate = template.Must(template.New("account").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{if .Title}}{{.Title}}{{else}}Learning material account{{end}}</title>
<style>
*{box-sizing:border-box}body{margin:0;min-height:100vh;display:grid;place-items:center;padding:24px;background:#f7f8fa;color:#172033;font:15px/1.45 system-ui,sans-serif}.card{width:min(100%,420px);background:#fff;border:1px solid #d8dee8;border-radius:12px;padding:28px;box-shadow:0 8px 30px #17203312}h1{font-size:24px;margin:0 0 7px}p{color:#667085;margin:0 0 20px}label{display:block;font-weight:650;margin:13px 0 5px}input{width:100%;padding:10px 11px;border:1px solid #98a2b3;border-radius:5px;font:inherit}button{width:100%;margin-top:20px;padding:10px;border:1px solid #175cd3;border-radius:5px;background:#175cd3;color:#fff;font:inherit;font-weight:700;cursor:pointer}.error{padding:10px 12px;border:1px solid #f0b4ae;background:#fff6f5;color:#b42318;border-radius:5px;margin:14px 0}.links{display:flex;justify-content:space-between;gap:12px;margin-top:18px;font-size:14px}.links a,.single-link a{color:#175cd3}.single-link{margin-top:20px}
</style></head><body><main class="card">
{{if eq .Page "create"}}
<h1>Create account</h1><p>Your email address is your username.</p>{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post"><label for="email">Email address</label><input id="email" name="email" type="email" autocomplete="email" required value="{{.Email}}"><label for="password">Password</label><input id="password" name="password" type="password" autocomplete="new-password" minlength="10" required><button type="submit">Create account</button></form><div class="single-link"><a href="/account/sign-in">Already have an account? Sign in</a></div>
{{else if eq .Page "sign-in"}}
<h1>Sign in</h1><p>Use your email address and password.</p>{{if .Error}}<div class="error">{{.Error}}</div>{{end}}
<form method="post"><input type="hidden" name="next" value="{{.Next}}"><label for="email">Email address</label><input id="email" name="email" type="email" autocomplete="email" required autofocus value="{{.Email}}"><label for="password">Password</label><input id="password" name="password" type="password" autocomplete="current-password" required><button type="submit">Sign in</button></form><div class="links"><a href="/account/create">Create account</a><a href="/account/forgot-password">Forgot password?</a></div>
{{else if eq .Page "forgot"}}
<h1>Reset password</h1><p>Enter your account email address.</p><form method="post"><label for="email">Email address</label><input id="email" name="email" type="email" autocomplete="email" required autofocus><button type="submit">Send reset link</button></form><div class="single-link"><a href="/account/sign-in">Back to sign in</a></div>
{{else if eq .Page "reset"}}
<h1>Choose a new password</h1><p>Use at least 10 characters.</p>{{if .Error}}<div class="error">{{.Error}}</div>{{end}}<form method="post"><input type="hidden" name="token" value="{{.Token}}"><label for="password">New password</label><input id="password" name="password" type="password" autocomplete="new-password" minlength="10" required autofocus><button type="submit">Update password</button></form>
{{else}}
<h1>{{.Title}}</h1><p>{{.Message}}</p><div class="single-link"><a href="/account/sign-in">Go to sign in</a></div>
{{end}}
</main></body></html>`))
