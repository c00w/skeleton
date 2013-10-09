package common

import (
	"io"
	"net/http"
	"strings"
)

type HttpAPI struct {
	ip string
}

// function to initialize new http struct
func NewHttpClient(ip string) (h *HttpAPI) {
	h = &HttpAPI{ip}
	return
}

// Post function to clean up http.Post calls in code, method for http struct
func (h *HttpAPI) Post(url string, content string, b io.Reader) (resp *http.Response, err error) {
	c := MakeHttpClient()

	resp, err = c.Post("http://"+h.ip+"/"+url,
		content, b)

	return
}

// Post function to clean up http.Post calls in code, method for http struct
func (h *HttpAPI) Put(url string, content string, b io.Reader) (resp *http.Response, err error) {
	c := MakeHttpClient()

	req, err := http.NewRequest("PUT",
		"http://"+h.ip+"/"+url, b)
	if err != nil {
		return
	}

	resp, err = c.Do(req)

	return
}

// Post with a dictionary of header values
func (h *HttpAPI) PostHeader(url string, content string, b io.Reader, header http.Header) (resp *http.Response, err error) {
	c := MakeHttpClient()

	req, err := http.NewRequest("POST",
		"http://"+h.ip+"/"+url, b)
	if err != nil {
		return
	}

	req.Header = header
	req.Header.Set("Content-Type", content)

	resp, err = c.Do(req)
	return
}

// Get function to clean up http.Get calls in code, method for http struct
func (h *HttpAPI) Get(url string) (resp *http.Response, err error) {
	c := MakeHttpClient()

	resp, err = c.Get("http://" + h.ip + "/" + url)

	return
}

// Delete function to clean up http.NewRequest("DELETE"...) call, method for http struct
func (h *HttpAPI) Delete(url string) (resp *http.Response, err error) {
	c := MakeHttpClient()
	b := strings.NewReader("")

	req, err := http.NewRequest("DELETE",
		"http://"+h.ip+"/"+url, b)
	if err != nil {
		return
	}
	resp, err = c.Do(req)
	return
}
