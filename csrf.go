package main

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"
)

const (
	csrfCookieName = "tyvm_csrf"
	csrfFormField  = "_csrf"
)

type ctxKey int

const csrfCtxKey ctxKey = 0

func newCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand failures are catastrophic; panic so we don't issue a predictable token.
		panic("csrf: rand.Read failed: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// csrfToken returns the token associated with the current request, suitable
// for embedding in a form. Returns an empty string when no token is set
// (which should not happen if csrfMiddleware is in the chain).
func csrfToken(r *http.Request) string {
	if v, ok := r.Context().Value(csrfCtxKey).(string); ok {
		return v
	}
	return ""
}

// csrfMiddleware implements the double-submit-cookie pattern: every request
// gets (or keeps) a token cookie, and unsafe methods must echo the same token
// back in the `_csrf` form field.
func csrfMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := ""
		if c, err := r.Cookie(csrfCookieName); err == nil && c.Value != "" {
			token = c.Value
		}
		if token == "" {
			token = newCSRFToken()
			http.SetCookie(w, &http.Cookie{
				Name:     csrfCookieName,
				Value:    token,
				Path:     "/",
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}

		if isUnsafeMethod(r.Method) {
			submitted := r.FormValue(csrfFormField)
			if submitted == "" ||
				subtle.ConstantTimeCompare([]byte(submitted), []byte(token)) != 1 {
				http.Error(w, "Invalid CSRF token", http.StatusForbidden)
				return
			}
		}

		ctx := context.WithValue(r.Context(), csrfCtxKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func isUnsafeMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}
