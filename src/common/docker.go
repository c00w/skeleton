package common

import (
	"bytes"
	"encoding/base64"
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

type HttpAPI struct {
	ip string
}

type Docker struct {
	h *HttpAPI
}

// function to initialize new http struct
func NewHttpClient(ip string) (h *HttpAPI) {
	h = &HttpAPI{ip}
	return
}

// function to initialize new docker struct
func NewDocker(ip string) (D *Docker) {
	D = &Docker{NewHttpClient(ip)}
	return
}

// Post function to clean up http.Post calls in code, method for http struct
func (h *HttpAPI) Post(url string, content string, b io.Reader) (resp *http.Response, err error) {
	c := MakeHttpClient()

	resp, err = c.Post("http://"+h.ip+":4243/"+url,
		content, b)

	return
}

// Post with a dictionary of header values
func (h *HttpAPI) PostHeader(url string, content string, b io.Reader, header http.Header) (resp *http.Response, err error) {
	c := MakeHttpClient()

	req, err := http.NewRequest("POST",
		"http://"+h.ip+":4243/"+url, b)
	if err != nil {
		log.Fatal(err)
	}

	req.Header = header
	req.Header.Set("Content-TYpe", content)

	resp, err = c.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	return
}

// Get function to clean up http.Get calls in code, method for http struct
func (h *HttpAPI) Get(url string) (resp *http.Response, err error) {
	c := MakeHttpClient()

	resp, err = c.Get("http://" + h.ip + ":4243/" + url)

	return
}

// Delete function to clean up http.NewRequest("DELETE"...) call, method for http struct
func (h *HttpAPI) Delete(url string) (resp *http.Response, err error) {
	c := MakeHttpClient()

	req, err := http.NewRequest("DELETE",
		"http://"+h.ip+":4243/"+url, nil)
	if err != nil {
		log.Fatal(err)
	}
	resp, err = c.Do(req)
	if err != nil {
		log.Fatal(err)
	}

	return
}

func (D *Docker) GetIP() string {
	return D.h.ip
}

// InspectContainer take a container id and returns its port its info
func (D *Docker) InspectContainer(id string) (info *containerInfo, err error) {
	i := &containerInfo{}

	resp, err := D.h.Get("/containers/" + id + "/json")
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

// runImage takes a docker image to run, and makes sure it is running
func (D *Docker) RunImage(imagename string, env []string) (id string, err error) {

	c := make(map[string]interface{})
	c["Image"] = imagename
	c["Env"] = env

	ba, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	b := bytes.NewBuffer(ba)

	resp, err := D.h.Post("containers/create", "application/json", b)

	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		msg := fmt.Sprintf("Create Container Response status is %d", resp.StatusCode)
		return "", errors.New(msg)
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
	resp, err = D.h.Post("containers/"+id+"/start", "application/json", b)

	defer resp.Body.Close()

	LogReader(resp.Body)

	if resp.StatusCode != 204 {
		msg := fmt.Sprintf("Start Container Response status is %d", resp.StatusCode)
		return "", errors.New(msg)
	}

	log.Printf("Container running")
	return id, nil

}

// Stopcontainer stops a container
func (D *Docker) StopContainer(container string) (err error) {
	log.Print("Stopping container ", container)
	b := bytes.NewBuffer(nil)

	resp, err := D.h.Post("containers/"+container+"/stop?t=1", "application/json", b)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	LogReader(resp.Body)
	return nil
}

// StopImage takes a image to stop and stops it
func (D *Docker) StopImage(imagename string) {
	log.Print("Stopping image ", imagename)

	running, id, err := D.ImageRunning(imagename)
	if err != nil {
		log.Fatal(err)
	}

	if running {
		err = D.StopContainer(id)
		if err != nil {
			log.Fatal(err)
		}
		D.DeleteContainer(id)
	}
}

func (D *Docker) DeleteContainer(id string) (err error) {
	log.Print("deleting container ", id)
	resp, err := D.h.Delete("containers/" + id)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	LogReader(resp.Body)
	return
}

// TagImage tags an already existing image in the repository
func (D *Docker) TagImage(name string, tag string) (err error) {
	b := bytes.NewBuffer(nil)

	resp, err := D.h.Post("images/"+name+"/tag?repo="+tag+"&force=1", "application/json", b)

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
func (D *Docker) PushImage(w io.Writer, name string) (err error) {
	b := bytes.NewBuffer(nil)

	auth := base64.StdEncoding.EncodeToString([]byte("{}"))
	header := make(http.Header)
	header.Set("X-Registry-Auth", auth)

	url := "images/" + name + "/push"

	resp, err := D.h.PostHeader(url,
		"application/json", b, header)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {

		_, err = io.Copy(w, resp.Body)
		if err != nil {
			io.WriteString(w, err.Error())
		}

		io.WriteString(w, resp.Header.Get("Location")+"\n")
		io.WriteString(w, url+"\n")

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
func (D *Docker) BuildImage(fd io.Reader, name string) (err error) {

	v := fmt.Sprintf("%d", time.Now().Unix())
	resp, err := D.h.Post("build?t="+name+"%3A"+v,
		"application/tar", fd)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	LogReader(resp.Body)

	return D.TagImage(name+"%3A"+v, name)
}

// loadImage pulls a specified image into a docker instance
func (D *Docker) LoadImage(imagename string) (err error) {
	b := bytes.NewBuffer(nil)
	resp, err := D.h.Post("images/create?fromImage="+imagename,
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
func (D *Docker) ListContainers() (c []containerInfo, err error) {
	resp, err := D.h.Get("containers/json")

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
func (D *Docker) ImageRunning(imagename string) (running bool, id string, err error) {

	containers, err := D.ListContainers()
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
