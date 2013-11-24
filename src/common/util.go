package common

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
    "crypto/tls"
)

var hc *http.Client = nil

func MakeHttpClient() *http.Client {
	if hc == nil {
        tr := &http.Transport{
            TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
        }
		hc = &http.Client{Transport: tr}
	}
	return hc
}

type Multiplexer struct {
    writers map[io.Writer]bool
}

func NewMultiplexer() *Multiplexer {
    m := &Multiplexer{}
    m.writers = make(map[io.Writer]bool)
    m.Attach(os.Stdout)
    return m
}

func (m *Multiplexer) Attach(w io.Writer) {
    m.writers[w] = true
}
func (m *Multiplexer) Detach(w io.Writer) {
    delete(m.writers, w)
}
func (m *Multiplexer) Write(p []byte) (n int, err error) {
    for w := range(m.writers) {
        w.Write(p)
    }
    n = len(p)
    err = nil
    return
}
type EncWriter struct {
	encoder *json.Encoder
	buffer []byte
}

func NewEncWriter(w io.Writer) *EncWriter {
	writer := new(EncWriter)
	writer.encoder = json.NewEncoder(w)
	return writer
}

func (enc *EncWriter) Write(p []byte) (n int, err error) {
    n = len(p)
    buffer := enc.buffer
    buffer = append(buffer, p...)
    sbuffer := string(buffer)
    for strings.Contains(sbuffer, "\n") {
        index := strings.Index(sbuffer, "\n")
        enc.Log(sbuffer[0:index])
        sbuffer = sbuffer[index+1:]
    }
    enc.buffer = []byte(sbuffer)
    return n,nil
}

func (enc *EncWriter) Log(s string) {
	enc.encoder.Encode(Message{Message_type: "message", Message: s})
}

func (enc *EncWriter) SetError(err error) {
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
