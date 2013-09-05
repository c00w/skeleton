package main

import (
	"common"
	"io"
	"log"
	"net/http"
)

func handleImage(w http.ResponseWriter, r *http.Request) {
	tag := r.URL.Query()["name"]
	if len(tag) > 0 {
		common.BuildImage("localhost", r.Body, tag[0])
	}
}

func main() {

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "0")
	})

	http.HandleFunc("/image", handleImage)

	http.HandleFunc("/deploy", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "{}")
	})

	log.Fatal(http.ListenAndServe(":900", nil))
}
