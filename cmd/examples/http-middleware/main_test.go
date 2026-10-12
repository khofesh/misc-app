package main

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
})

func TestServerHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	serverHeader(okHandler).ServeHTTP(rec, req)

	if got := rec.Header().Get("Server"); got != "Go" {
		t.Errorf("Server header = %q, want %q", got, "Go")
	}
	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want next handler to run", rec.Body.String())
	}
}

func TestLogRequest(t *testing.T) {
	var buf bytes.Buffer
	orig := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(orig) })

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/some/path?q=1", nil)
	req.RemoteAddr = "10.0.0.1:1234"

	logRequest(okHandler).ServeHTTP(rec, req)

	if rec.Body.String() != "ok" {
		t.Errorf("body = %q, want next handler to run", rec.Body.String())
	}
	logged := buf.String()
	for _, want := range []string{
		"request received",
		"user.ip=10.0.0.1:1234",
		"request.method=GET",
		"request.url=\"/some/path?q=1\"",
		"request.proto=HTTP/1.1",
	} {
		if !strings.Contains(logged, want) {
			t.Errorf("log %q does not contain %q", logged, want)
		}
	}
}

func TestRequireBasicAuthentication(t *testing.T) {
	tests := []struct {
		name       string
		setAuth    bool
		username   string
		password   string
		wantStatus int
	}{
		{"no credentials", false, "", "", http.StatusUnauthorized},
		{"wrong username", true, "root", "secret", http.StatusUnauthorized},
		{"wrong password", true, "admin", "wrong", http.StatusUnauthorized},
		{"empty credentials", true, "", "", http.StatusUnauthorized},
		{"valid credentials", true, "admin", "secret", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/admin", nil)
			if tt.setAuth {
				req.SetBasicAuth(tt.username, tt.password)
			}

			requireBasicAuthentication(okHandler).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantStatus == http.StatusUnauthorized {
				if got := rec.Header().Get("WWW-Authenticate"); got != `Basic realm="protected"` {
					t.Errorf("WWW-Authenticate = %q", got)
				}
				if rec.Body.String() == "ok" {
					t.Error("next handler ran for an unauthorized request")
				}
			} else if rec.Body.String() != "ok" {
				t.Errorf("body = %q, want next handler to run", rec.Body.String())
			}
		})
	}
}

func TestHandlers(t *testing.T) {
	tests := []struct {
		name    string
		handler http.HandlerFunc
		want    string
	}{
		{"home", home, "Welcome to the home page!"},
		{"admin", admin, "Admin dashboard - you are authenticated!"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			tt.handler(rec, httptest.NewRequest(http.MethodGet, "/", nil))

			if rec.Code != http.StatusOK {
				t.Errorf("status = %d, want 200", rec.Code)
			}
			body, _ := io.ReadAll(rec.Body)
			if string(body) != tt.want {
				t.Errorf("body = %q, want %q", body, tt.want)
			}
		})
	}
}
