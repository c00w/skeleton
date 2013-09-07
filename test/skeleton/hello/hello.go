package main

import (
	"io"
	"log"
	"net/http"
)

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world")
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}
