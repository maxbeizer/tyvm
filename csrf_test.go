package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) (*httptest.Server, *App) {
	t.Helper()
	app := newTestApp(t)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /tanks", app.createTankHandler)
	srv := httptest.NewServer(csrfMiddleware(mux))
	t.Cleanup(srv.Close)
	return srv, app
}

func TestCSRF_RejectsPostWithoutToken(t *testing.T) {
	srv, _ := newTestServer(t)

	resp, err := http.PostForm(srv.URL+"/tanks", url.Values{"name": {"x"}})
	if err != nil {
		t.Fatalf("PostForm: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected 403, got %d: %s", resp.StatusCode, body)
	}
}

func TestCSRF_AcceptsPostWithMatchingToken(t *testing.T) {
	srv, _ := newTestServer(t)

	// First do a GET-equivalent (any safe method) to obtain the cookie token.
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Jar: nil,
	}

	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/tanks", nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("priming GET: %v", err)
	}
	resp.Body.Close()

	var token string
	for _, c := range resp.Cookies() {
		if c.Name == csrfCookieName {
			token = c.Value
		}
	}
	if token == "" {
		t.Fatal("no CSRF cookie issued")
	}

	form := url.Values{"name": {"Reef"}, csrfFormField: {token}}
	req, _ = http.NewRequest(http.MethodPost, srv.URL+"/tanks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: token})
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusSeeOther {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("expected 303 redirect, got %d: %s", resp.StatusCode, body)
	}
}

func TestCSRF_RejectsMismatchedToken(t *testing.T) {
	srv, _ := newTestServer(t)

	form := url.Values{"name": {"Reef"}, csrfFormField: {"wrong-token"}}
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/tanks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: csrfCookieName, Value: "real-token"})

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403, got %d", resp.StatusCode)
	}
}
