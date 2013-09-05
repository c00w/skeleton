package common

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
)

type idDump struct {
	Id string
}

// runContainer takes a ip and a docker image to run, and makes sure it is running
func RunContainer(ip string, image string) {
	h := MakeHttpClient()
	b := bytes.NewBuffer([]byte("{\"Image\":\"" + image + "\"}"))
	resp, err := h.Post("http://"+ip+":4243/containers/create",
		"application/json", b)
	defer resp.Body.Close()

	s, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		log.Fatal("response status code not 201")
	}

	if err != nil {
		log.Fatal(err)
	}

	a := new(idDump)
	json.Unmarshal(s, a)
	id := a.Id

	log.Printf("Container created id:%s", id)

	b = bytes.NewBuffer([]byte("{}"))
	resp, err = h.Post("http://"+ip+":4243/containers/"+id+"/start",
		"application/json", b)
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		LogReader(resp.Body)
		log.Fatal("start status code is not 204 it is %i", resp.StatusCode)

	}

	log.Printf("Container running")

}

// buildImage takes a tarfile, and builds it
func BuildImage(ip string, fd io.Reader, name string) {

	h := MakeHttpClient()
	resp, err := h.Post("http://"+ip+":4243/build?t="+name,
		"application/tar", fd)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	LogReader(resp.Body)
	log.Print(resp.StatusCode)
}

// loadImage pulls a specified image into a docker instance
func LoadImage(ip string, image string) {
	h := MakeHttpClient()
	b := bytes.NewBuffer(nil)
	resp, err := h.Post("http://"+ip+":4243/images/create?fromImage="+image,
		"text", b)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	LogReader(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("create status code is not 200 %s", resp.StatusCode)
	}

	log.Printf("Image fetched %s", image)
}
