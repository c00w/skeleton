package common

import (
	"io"
	"log"
	"net/http"
	"encoding/json"
	"errors"
)

var hc *http.Client = nil

func MakeHttpClient() *http.Client {
	if hc == nil {
		hc = &http.Client{}
	}
	return hc
}

type Message struct {
    Message_type string
    Status string
    Message string
    }

func JsonReader(r io.Reader) (error) {
    dec := json.NewDecoder(r)
    m := &Message{}
    //infinite loop
    for {
        err := dec.Decode(m)
        if(err!=nil) {
            return nil
        } else if(m.Message_type=="error") {
            //log.Print(m.message)
            return errors.New(m.Message)
        } else {
            log.Print(m.Message)
        }
    }
}
    
func LogReader(r io.Reader) {
	buff := make([]byte, 1024)
	for n, err := r.Read(buff); err == nil; n, err = r.Read(buff) {
		log.Print(string(buff[0:n]))
	}
}
