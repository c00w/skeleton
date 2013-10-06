package main

import (
	"io"
	"log"
	"net/http"
	"runtime"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "hello world\n")
	})

	log.Fatal(http.ListenAndServe(":80", nil))
}
