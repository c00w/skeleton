package main

import (
	"io"
	"log"
	"net/http"
)

func main() {

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "gatekeeper v0")
	})

	log.Fatal(http.ListenAndServe(":800", nil))
}
