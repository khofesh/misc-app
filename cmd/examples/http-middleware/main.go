// https://www.alexedwards.net/blog/making-and-using-middleware
package main

import (
	"log"
	"net/http"
)

func messageHandler(message string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(message))
	})
}

func main() {
	mux := http.NewServeMux()

	mux.Handle("GET /", messageHandler("hello world"))

	log.Print("listening on :3000...")
	err := http.ListenAndServe(":3000", mux)
	log.Fatal(err)
}
