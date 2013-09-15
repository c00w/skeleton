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

type DockerInfo struct {
	Ip         string
	Containers []containerInfo
	Images     []imageInfo
	Updated    time.Time
}

type containerInfo struct {
	Id              string
	Image           string
	NetworkSettings struct {
		PortMapping struct {
			Tcp map[string]string
		}
	}
}

type imageInfo struct {
	Id         string
	Repository string
}

// InspectContainer take a ip and a container id and returns its port its info
func InspectContainer(ip string, id string) (info *containerInfo, err error) {
	h := MakeHttpClient()
	i := &containerInfo{}

	resp, err := h.Get("http://" + ip + ":4243/containers/" + id + "/json")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return i, errors.New("Inspect Container Status is not 200")
	}

	rall, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	json.Unmarshal(rall, i)

	log.Print("Container inspected")
	return i, nil
}

// runImage takes a ip and a docker image to run, and makes sure it is running
func RunImage(ip string, imagename string, hint bool) (id string, err error) {
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
		return "", err
	}
	b := bytes.NewBuffer(ba)
	resp, err := h.Post("http://"+ip+":4243/containers/create",
		"application/json", b)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		return "", errors.New("response status code not 201")
	}

	s, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	a := &containerInfo{}
	json.Unmarshal(s, a)
	id = a.Id

	log.Printf("Container created id:%s", id)

	b = bytes.NewBuffer([]byte("{}"))
	resp, err = h.Post("http://"+ip+":4243/containers/"+id+"/start",
		"application/json", b)
	defer resp.Body.Close()

	LogReader(resp.Body)

	if resp.StatusCode != 204 {
		return "", errors.New("start status code is not 204")
	}

	log.Printf("Container running")
	return id, nil

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

// PushImage pushes an image to a docker index
func PushImage(ip string, w io.Writer, name string) (err error) {
	h := MakeHttpClient()
	b := bytes.NewBuffer([]byte("{}"))

	resp, err := h.Post("http://"+ip+":4243/images/"+name+"/push",
		"application/json", b)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {

		_, err = io.Copy(w, resp.Body)
		s := fmt.Sprintf("Push image status code is not 200 it is %d",
			resp.StatusCode)
		return errors.New(s)
	}

	defer resp.Body.Close()
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

// buildImage takes a tarfile, and builds it
func BuildImage(ip string, fd io.Reader, name string) (err error) {

	h := MakeHttpClient()
	v := fmt.Sprintf("%d", time.Now().Unix())
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

// ListContainers gives the state for a specific docker container
func ListContainers(ip string) (c []containerInfo, err error) {
	h := MakeHttpClient()
	resp, err := h.Get("http://" + ip + ":4243/containers/json")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return c, errors.New("Response code is not 200")
	}

	message, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	containers := &c
	err = json.Unmarshal(message, containers)
	return
}

// ImageRunning states whether an image is running on a docker instance
func ImageRunning(ip string, imagename string) (running bool, id string, err error) {

	containers, err := ListContainers(ip)
	if err != nil {
		return false, "", err
	}

	for _, v := range containers {
		if strings.SplitN(v.Image, ":", 2)[0] == imagename {
			return true, v.Id, nil
		}
	}
	return
}
