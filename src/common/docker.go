package common

import (
	"bytes"
	"encoding/json"
	"errors"
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

// runImage takes a ip and a docker image to run, and makes sure it is running
func RunImage(ip string, imagename string, hint bool) (err error) {
	h := MakeHttpClient()

	e := make([]string, 1)
	e[0] = "HOST=" + ip

	c := make(map[string]interface{})
	c["Image"] = imagename
	if hint {
		c["Env"] = e
	}

	ba, err := json.Marshal(c)
	if err != nil {
		return err
	}
	b := bytes.NewBuffer(ba)
	resp, err := h.Post("http://"+ip+":4243/containers/create",
		"application/json", b)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return errors.New("response status code not 201")
	}

	s, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
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
		return errors.New("start status code is not 204")
	}

	LogReader(resp.Body)
	log.Printf("Container running")
	return nil

}

// Stopcontainer stops a container
func StopContainer(ip string, container string) (err error) {
	log.Print("Stopping container ", container)
	h := MakeHttpClient()
	b := bytes.NewBuffer(nil)

	resp, err := h.Post("http://"+ip+":4243/containers/"+container+"/stop?t=1",
		"application/json", b)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	LogReader(resp.Body)
	return nil
}

// StopImage takes a ip and a image to stop and stops it
func StopImage(ip string, imagename string) {
	log.Print("Stopping image ", imagename)

	running, id, err := ImageRunning(ip, imagename)
	if err != nil {
		log.Fatal(err)
	}

	if running {
		err = StopContainer(ip, id)
		if err != nil {
			log.Fatal(err)
		}
		DeleteContainer(ip, id)
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
func TagImage(ip string, name string, tag string) (err error) {
	h := MakeHttpClient()
	b := bytes.NewBuffer(nil)

	resp, err := h.Post("http://"+ip+":4243/images/"+name+"/tag?repo="+tag+"&force=1",
		"application/json", b)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		log.Print(resp)
		return errors.New("Code not 201")
	}

	return nil

}

// buildImage takes a tarfile, and builds it
func BuildImage(ip string, fd io.Reader, name string) (err error) {

	h := MakeHttpClient()
	v := fmt.Sprintf("%d", time.Now().Unix())
	log.Print(v)
	resp, err := h.Post("http://"+ip+":4243/build?t="+name+"%3A"+v,
		"application/tar", fd)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	LogReader(resp.Body)

	return TagImage(ip, name+"%3A"+v, name)
}

// loadImage pulls a specified image into a docker instance
func LoadImage(ip string, imagename string) (err error) {
	h := MakeHttpClient()
	b := bytes.NewBuffer(nil)
	resp, err := h.Post("http://"+ip+":4243/images/create?fromImage="+imagename,
		"text", b)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	LogReader(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("create status code is not 200 %s", resp.StatusCode)
	}

	log.Printf("Image fetched %s", imagename)
	return nil
}

// ImageRunning states whether an image is running on a docker instance
func ImageRunning(ip string, imagename string) (running bool, id string, err error) {
	h := MakeHttpClient()
	resp, err := h.Get("http://" + ip + ":4243/containers/json")

	if err != nil {
		return false, "", err
	}
	defer resp.Body.Close()

	message, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return false, "", err
	}

	containers := new([]container)
	json.Unmarshal(message, containers)

	for _, v := range *containers {
		if strings.SplitN(v.Image, ":", 2)[0] == imagename {
			return true, v.Id, nil
		}
	}
	return false, "", nil
}
