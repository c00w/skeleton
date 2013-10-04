package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world\n")
		io.WriteString(w, os.Getenv("GATEKEEPER"))
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}
