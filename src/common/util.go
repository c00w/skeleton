package common

import (
	"encoding/json"
	"errors"
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

type EncWriter struct {
	encoder *json.Encoder
}

func NewEncWriter(w io.Writer) *EncWriter {
	writer := new(EncWriter)
	writer.encoder = json.NewEncoder(w)
	return writer
}

func (enc *EncWriter) Write(s string) {
	enc.encoder.Encode(Message{Message_type: "message", Message: s})
}

func (enc *EncWriter) ErrWrite(err error) {
	enc.encoder.Encode(Message{Message_type: "error",
		Status: "500", Message: err.Error()})
}

type Message struct {
	Message_type string
	Status       string
	Message      string
}

func JsonReader(r io.Reader) error {
	dec := json.NewDecoder(r)
	m := &Message{}
	for {
		err := dec.Decode(m)
		if err != nil {
			return nil
		} else if m.Message_type == "error" {
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
