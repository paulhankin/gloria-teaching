package account

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"learningmaterial/internal/store"
)

type sentMail struct {
	to, subject, body string
}

type fakeMailer struct{ messages []sentMail }

func (m *fakeMailer) Send(to, subject, body string) error {
	m.messages = append(m.messages, sentMail{to, subject, body})
	return nil
}

func testServer(t *testing.T, allowed ...string) (*httptest.Server, *fakeMailer) {
	t.Helper()
	db, err := store.Open(filepath.Join(t.TempDir(), "accounts.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	mailer := &fakeMailer{}
	manager := New(db, []byte("a sufficiently long test secret"), allowed, mailer, "")
	mux := http.NewServeMux()
	manager.Register(mux)
	mux.Handle("/", manager.RequireAccess(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome " + Email(r)))
	})))
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	return server, mailer
}

func postForm(t *testing.T, client *http.Client, endpoint string, values url.Values) *http.Response {
	t.Helper()
	resp, err := client.PostForm(endpoint, values)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func responseText(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func tokenFromMail(t *testing.T, body string) string {
	t.Helper()
	for _, field := range strings.Fields(body) {
		if strings.Contains(field, "?token=") {
			u, err := url.Parse(field)
			if err != nil {
				t.Fatal(err)
			}
			return u.Query().Get("token")
		}
	}
	t.Fatal("mail has no token link")
	return ""
}

func TestAllowedAccountCreationVerificationAndSignIn(t *testing.T) {
	server, mailer := testServer(t, "allowed@example.com")
	client := server.Client()
	resp := postForm(t, client, server.URL+"/account/create", url.Values{
		"email": {"allowed@example.com"}, "password": {"long-enough-password"},
	})
	if body := responseText(t, resp); !strings.Contains(body, "Check your email") {
		t.Fatalf("create response = %s", body)
	}
	if len(mailer.messages) != 1 || mailer.messages[0].to != "allowed@example.com" {
		t.Fatalf("messages = %#v", mailer.messages)
	}
	token := tokenFromMail(t, mailer.messages[0].body)
	resp, err := client.Get(server.URL + "/account/verify?token=" + url.QueryEscape(token))
	if err != nil {
		t.Fatal(err)
	}
	if body := responseText(t, resp); !strings.Contains(body, "Email confirmed") {
		t.Fatalf("verify response = %s", body)
	}

	// httptest is HTTP while production is HTTPS, so preserve the Secure cookie manually.
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	resp = postForm(t, client, server.URL+"/account/sign-in", url.Values{
		"email": {"allowed@example.com"}, "password": {"long-enough-password"}, "next": {"/private"},
	})
	if resp.StatusCode != http.StatusSeeOther || len(resp.Cookies()) == 0 {
		t.Fatalf("sign-in status/cookies = %d / %#v", resp.StatusCode, resp.Cookies())
	}
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/private", nil)
	req.AddCookie(resp.Cookies()[0])
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if body := responseText(t, resp); body != "welcome allowed@example.com" {
		t.Fatalf("protected response = %q", body)
	}
}

func TestAccountOutsideAllowlistCannotAccessSite(t *testing.T) {
	server, mailer := testServer(t, "allowed@example.com")
	client := server.Client()
	resp := postForm(t, client, server.URL+"/account/create", url.Values{
		"email": {"other@example.com"}, "password": {"long-enough-password"},
	})
	if body := responseText(t, resp); !strings.Contains(body, "access to the site has not been granted") {
		t.Fatalf("create response = %s", body)
	}
	if len(mailer.messages) != 0 {
		t.Fatalf("unexpected mail = %#v", mailer.messages)
	}
	resp = postForm(t, client, server.URL+"/account/sign-in", url.Values{
		"email": {"other@example.com"}, "password": {"long-enough-password"},
	})
	if body := responseText(t, resp); !strings.Contains(body, "not currently allowed") {
		t.Fatalf("sign-in response = %s", body)
	}
}

func TestPasswordReset(t *testing.T) {
	server, mailer := testServer(t, "allowed@example.com")
	client := server.Client()
	resp := postForm(t, client, server.URL+"/account/create", url.Values{
		"email": {"allowed@example.com"}, "password": {"long-enough-password"},
	})
	responseText(t, resp)
	verifyToken := tokenFromMail(t, mailer.messages[0].body)
	resp, _ = client.Get(server.URL + "/account/verify?token=" + url.QueryEscape(verifyToken))
	responseText(t, resp)

	resp = postForm(t, client, server.URL+"/account/forgot-password", url.Values{"email": {"allowed@example.com"}})
	responseText(t, resp)
	resetToken := tokenFromMail(t, mailer.messages[1].body)
	resp = postForm(t, client, server.URL+"/account/reset-password", url.Values{
		"token": {resetToken}, "password": {"replacement-password"},
	})
	if body := responseText(t, resp); !strings.Contains(body, "Password updated") {
		t.Fatalf("reset response = %s", body)
	}

	client.CheckRedirect = func(req *http.Request, via []*http.Request) error { return http.ErrUseLastResponse }
	resp = postForm(t, client, server.URL+"/account/sign-in", url.Values{
		"email": {"allowed@example.com"}, "password": {"replacement-password"},
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("new password sign-in status = %d, body %s", resp.StatusCode, responseText(t, resp))
	}
}
