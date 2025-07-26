// https://www.alexedwards.net/blog/making-and-using-middleware
package main

import (
	"log"
	"net/http"
)

func middlewareOne(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL.Path, "executing middlewareOne")
		next.ServeHTTP(w, r)
		log.Println(r.URL.Path, "executing middlewareOne again")
	})
}

func fooHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path, "executing fooHandler")
	w.Write([]byte("foo"))
}

func barHandler(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL.Path, "executing barHandler")
	w.Write([]byte("bar"))
}

func main() {
	mux := http.NewServeMux()

	mux.Handle("GET /foo", http.HandlerFunc(fooHandler))
	mux.Handle("GET /bar", http.HandlerFunc(barHandler))

	log.Print("listening on :3000...")
	err := http.ListenAndServe(":3000", middlewareOne(mux))
	log.Fatal(err)
}
