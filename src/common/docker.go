package common

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type container struct {
	Id    string
	Image string
}

type image struct {
	Id         string
	Repository string
}

// runContainer takes a ip and a docker image to run, and makes sure it is running
func RunContainer(ip string, imagename string) {
	h := MakeHttpClient()
	b := bytes.NewBuffer([]byte("{\"Image\":\"" + imagename + "\"}"))
	resp, err := h.Post("http://"+ip+":4243/containers/create",
		"application/json", b)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	s, err := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		log.Print(string(s))
		log.Fatal("response status code not 201")
	}
	if err != nil {
		log.Fatal(err)
	}

	a := new(container)
	json.Unmarshal(s, a)
	id := a.Id

	log.Printf("Container created id:%s", id)

	b = bytes.NewBuffer([]byte("{}"))
	resp, err = h.Post("http://"+ip+":4243/containers/"+id+"/start",
		"application/json", b)
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		LogReader(resp.Body)
		log.Fatal("start status code is not 204 it is ", resp.StatusCode)

	}

	LogReader(resp.Body)
	log.Printf("Container running")

}

// Stopcontainer stops a container
func StopContainer(ip string, container string) {
	log.Print("Stopping container ", container)
	h := MakeHttpClient()
	b := bytes.NewBuffer(nil)

	resp, err := h.Post("http://"+ip+":4243/containers/"+container+"/stop?t=1",
		"application/json", b)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	LogReader(resp.Body)
}

// StopImage takes a ip and a image to stop and stops it
func StopImage(ip string, imagename string) {
	log.Print("Stopping image ", imagename)
	h := MakeHttpClient()

	resp, err := h.Get("http://" + ip + ":4243/containers/json")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	message, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	containers := new([]container)
	json.Unmarshal(message, containers)

	resp, err = h.Get("http://" + ip + ":4243/images/json")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	message, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	images := new([]image)
	json.Unmarshal(message, images)

	imagemap := make(map[string]string)

	for _, v := range *images {
		imagemap[v.Id] = v.Repository
	}

	for _, v := range *containers {
		if strings.SplitN(v.Image, ":", 2)[0] == imagename {
			id := v.Id
			StopContainer(ip, id)
			DeleteContainer(ip, id)
			break
		}
	}
}

func DeleteContainer(ip string, id string) {
	log.Print("deleting container ", id)
	h := MakeHttpClient()
	req, err := http.NewRequest("DELETE",
		"http://"+ip+":4243/containers/"+id, nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err := h.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	LogReader(resp.Body)
}

// TagImage tags an already existing image in the repository
func TagImage(ip string, name string, tag string) {
	h := MakeHttpClient()
	b := bytes.NewBuffer(nil)

	resp, err := h.Post("http://"+ip+":4243/images/"+name+"/tag?repo="+tag+"&force=1",
		"application/json", b)

	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		log.Print(resp)
		log.Fatal("Code not 201")
	}

}

// buildImage takes a tarfile, and builds it
func BuildImage(ip string, fd io.Reader, name string) {

	h := MakeHttpClient()
	v := fmt.Sprintf("%d", time.Now().Unix())
	log.Print(v)
	resp, err := h.Post("http://"+ip+":4243/build?t="+name+"%3A"+v,
		"application/tar", fd)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	LogReader(resp.Body)

	TagImage(ip, name+"%3A"+v, name)
}

// loadImage pulls a specified image into a docker instance
func LoadImage(ip string, imagename string) {
	h := MakeHttpClient()
	b := bytes.NewBuffer(nil)
	resp, err := h.Post("http://"+ip+":4243/images/create?fromImage="+imagename,
		"text", b)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	LogReader(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("create status code is not 200 %s", resp.StatusCode)
	}

	log.Printf("Image fetched %s", imagename)
}
