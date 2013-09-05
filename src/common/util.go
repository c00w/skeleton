package common

import (
	"io"
	"log"
	"net/http"
)

func MakeHttpClient() *http.Client {
	return &http.Client{}
}

func LogReader(r io.Reader) {
	buff := make([]byte, 1024)
	for _, err := r.Read(buff); err == nil; _, err = r.Read(buff) {
		log.Print(string(buff))
	}
}
