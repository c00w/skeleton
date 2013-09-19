package common

import (
	"io"
	"log"
	"net/http"
)

var hc *http.Client = nil

func MakeHttpClient() *http.Client {
	if hc == nil {
		hc = &http.Client{}
	}
	return hc
}

func LogReader(r io.Reader) {
	buff := make([]byte, 1024)
	for n, err := r.Read(buff); err == nil; n, err = r.Read(buff) {
		log.Print(string(buff[0:n]))
	}
}
