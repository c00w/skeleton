package main

import (
	"common"
	"io"
	"log"
	"net/http"
	"os"
)

func handleImage(w http.ResponseWriter, r *http.Request) {
	return
	tag := r.URL.Query()["name"]
	if len(tag) > 0 {
		common.BuildImage(os.Getenv("HOST"), r.Body, tag[0])
	}
	io.WriteString(w, "built")
}

func main() {

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "v0")
	})

	http.HandleFunc("/image", handleImage)

	http.HandleFunc("/deploy", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "deployed")
	})

	log.Fatal(http.ListenAndServe(":900", nil))
}
