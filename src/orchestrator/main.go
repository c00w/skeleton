package main

import (
	"io"
	"log"
	"net/http"
)

func main() {

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "0")
	})

	http.HandleFunc("/image", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "{}")
	})

	http.HandleFunc("/deploy", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "{}")
	})

	log.Fatal(http.ListenAndServe(":900", nil))
}
