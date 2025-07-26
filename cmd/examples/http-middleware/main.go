// https://www.alexedwards.net/blog/making-and-using-middleware
// https://www.alexedwards.net/blog/organize-your-go-middleware-without-dependencies
package main

import (
	"log/slog"
	"net/http"
	"os"
)

func serverHeader(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", "Go")
		next.ServeHTTP(w, r)
	})
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var (
			ip     = r.RemoteAddr
			method = r.Method
			url    = r.URL.String()
			proto  = r.Proto
		)

		userAttrs := slog.Group("user", "ip", ip)
		requestAttrs := slog.Group("request", "method", method, "url", url, "proto", proto)

		slog.Info("request received", userAttrs, requestAttrs)
		next.ServeHTTP(w, r)
	})
}

func requireBasicAuthentication(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		validUsername := "admin"
		validPassword := "secret"

		username, password, ok := r.BasicAuth()
		if !ok || username != validUsername || password != validPassword {
			w.Header().Set("WWW-Authenticate", `Basic realm="protected"`)
			http.Error(w, "401 Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func home(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Welcome to the home page!"))
}

func admin(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Admin dashboard - you are authenticated!"))
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /{$}", home)
	mux.Handle("GET /admin", requireBasicAuthentication(http.HandlerFunc(admin)))

	slog.Info("listening on :3000...")
	err := http.ListenAndServe(":3000", serverHeader(logRequest(mux)))
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
}
