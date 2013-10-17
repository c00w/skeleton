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

type Docker struct {
	h          *HttpAPI // contains ip string
	Containers []*Container
	Images     []*Image
	Updated    time.Time
}

type Container struct {
	Id              string
	Image           string
	D               *Docker
	NetworkSettings struct {
		PortMapping struct {
			Tcp map[string]string
		}
	}
	Volumes map[string]string
}

type Image struct {
	Id         string
	Tag        string
	Repository string
}

// function to initialize new Docker struct
func NewDocker(ip string) (D *Docker) {
	D = &Docker{}
	D.h = NewHttpClient(ip + ":4243")
	return
}

func (D *Docker) GetIP() string {
	return strings.Split(D.h.ip, ":")[0]
}

// function to periodically update information in Docker struct
func (D *Docker) Update() {
	for ; ; time.Sleep(60 * time.Second) {
		c, err := D.ListContainers()
		if err != nil {
			log.Print(err)
			continue
		}

		img, err := D.ListImages()
		if err != nil {
			log.Print(err)
			continue
		}

		D.Containers = c
		D.Images = img
		D.Updated = time.Now()
	}
}

// InspectContainer takes a container, and returns its port and its info
func (C *Container) Inspect() (err error) {

	resp, err := C.D.h.Get("containers/" + C.Id + "/json")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New("Inspect Container Status is not 200")
	}

	rall, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(rall, C)
	return
}

// runImage takes a docker image to run, and makes sure it is running
func (Img *Image) Run(D *Docker, imagename string, env []string) (C *Container, err error) {

	c := make(map[string]interface{})
	c["Image"] = imagename
	c["Env"] = env
	v := make(map[string]struct{})
	v["/foo"] = struct{}{}
	c["Volumes"] = v

	ba, err := json.Marshal(c)
	if err != nil {
		return
	}
	var b io.Reader
	b = bytes.NewBuffer(ba)

	resp, err := D.h.Post("containers/create", "application/json", b)

	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		msg := fmt.Sprintf("Create Container Response status is %d", resp.StatusCode)
		err = errors.New(msg)
		return
	}

	s, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(s, &C)
	C.D = D

	if err != nil {
		return
	}

	err = C.Start()

	return
}

func (C *Container) Start(binds []string) (err error) {

	log.Printf("Container created id:%s", C.Id)

	b := strings.NewReader("{\"Binds\":[\"/mnt:/foo\"]}")
	resp, err := C.D.h.Post("containers/"+C.Id+"/start", "application/json", b)

	defer resp.Body.Close()

	LogReader(resp.Body)

	if resp.StatusCode != 204 {
		msg := fmt.Sprintf("Start Container Response status is %d", resp.StatusCode)
		err = errors.New(msg)
		return
	}

	log.Printf("Container running")
	return

}

// StopContainer stops a container
func (C *Container) Stop() (err error) {
	log.Print("Stopping container ", C.Id)
	b := strings.NewReader("")

	log.Print(C.D)
	resp, err := C.D.h.Post("containers/"+C.Id+"/stop?t=1", "application/json", b)

	log.Print(resp.StatusCode)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	LogReader(resp.Body)
	return nil
}

// StopImage takes a image to stop and stops it
func (Img *Image) Stop(D *Docker, imagename string) {
	log.Print("Stopping image ", imagename)

	running, C, err := Img.IsRunning(D, imagename)
	if err != nil {
		log.Fatal(err)
	}

	if running {
		err = C.Stop()
		if err != nil {
			log.Fatal(err)
		}
		C.Delete()
	}
}

func (C *Container) Delete() (err error) {
	log.Print("deleting container ", C.Id)

	resp, err := C.D.h.Delete("containers/" + C.Id)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	LogReader(resp.Body)
	return
}

// TagImage tags an already existing image in the repository
func (Img *Image) AddTag(D *Docker, name string, tag string) (err error) {
	b := strings.NewReader("")

	resp, err := D.h.Post("images/"+name+"/tag?repo="+tag+"&force=1", "application/json", b)

	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		b, _ := ioutil.ReadAll(resp.Body)
		log.Print(resp)
		log.Print(string(b))
		return errors.New("Code not 201: " + resp.Status)
	}

	return nil

}

// PushImage pushes an image to a docker index
func (Img *Image) Push(D *Docker, w io.Writer, name string) (err error) {
	b := strings.NewReader("")

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
func (Img *Image) Build(D *Docker, fd io.Reader, name string) (err error) {

	v := fmt.Sprintf("%d", time.Now().Unix())
	resp, err := D.h.Post("build?t="+name+"%3A"+v,
		"application/tar", fd)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	LogReader(resp.Body)

	return Img.AddTag(D, name+"%3A"+v, name)
}

// loadImage pulls a specified image into a docker instance
func (Img *Image) Load(D *Docker, imagename string) (err error) {
	b := strings.NewReader("")
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
func (D *Docker) ListContainers() (c []*Container, err error) {
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
	err = json.Unmarshal(message, &c)
	for _, container := range c {
		container.D = D
	}
	return c, nil
}

// ListImages gives the state of the images for a specific Docker container
func (D *Docker) ListImages() (img []*Image, err error) {
	resp, err := D.h.Get("images/json")

	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return img, errors.New("Response code is not 200")
	}

	message, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	err = json.Unmarshal(message, &img)
	return
}

// ImageRunning states whether an image is running on a docker instance
func (Img *Image) IsRunning(D *Docker, imagename string) (running bool, C *Container, err error) {

	containers, err := D.ListContainers()
	if err != nil {
		return false, nil, err
	}

	for _, v := range containers {
		log.Print(v)
		if strings.SplitN(v.Image, ":", 2)[0] == imagename {
			return true, v, nil
		}
	}
	return
}
