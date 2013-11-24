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
	"net/url"
	"strings"
	"time"
)

type PortBinding struct {
	HostIp   string
	HostPort string
}

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
	Volumes      map[string]string
	Binds        []string
	PortBindings map[string][]PortBinding
}

type Image struct {
	Id         string
	Tag        string
	Repository string
	name       string
}

func NewImage(id string) (i *Image) {
	i = new(Image)
	i.Id = id
	return i
}

func NewNamedImage(name string) (i *Image) {
	i = new(Image)

	// Don't bother parsing things like 1.1.1.1:4243/foo:bar
	if len(strings.Split(name, "/")) > 1 {
		i.name = name
		return
	}

	if len(strings.Split(name, ":")) > 1 {
		i.Repository = strings.Split(name, ":")[0]
		i.Tag = strings.Split(name, ":")[1]
	} else {
		i.Repository = name
		i.Tag = "latest"
	}
	return i
}

func (I *Image) GetName() (name string) {
	if I.Id != "" {
		name = I.Id
		return
	}

	if I.Repository != "" {
		name = I.Repository + ":" + I.Tag
	} else {
		name = I.Tag
	}

	if name == "" {
		name = I.name
	}
	return
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
func (Img *Image) Run(D *Docker, env []string, port string) (C *Container, err error) {

	c := make(map[string]interface{})
	c["Image"] = Img.GetName()
	c["Env"] = env
	v := make(map[string]struct{})
	v["/foo"] = struct{}{}
	c["Volumes"] = v
	if len(port) > 0 {
		p := make(map[string]struct{})
		p[port] = struct{}{}
		c["ExposedPorts"] = p
	}

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
		LogReader(resp.Body)
		msg := fmt.Sprintf("Create Container Response status is %d", resp.StatusCode)
		err = errors.New(msg)
		return
	}

	s, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	C = &Container{}
	C.D = D

	err = json.Unmarshal(s, C)

	if err != nil {
		return
	}

	if len(port) > 0 {
		C.AddExposedPort(port)
	}

	C.AddBind("/mnt", "/foo")

	err = C.Start()

	return
}

func (C *Container) AddExposedPort(port string) {
	if C.PortBindings == nil {
		C.PortBindings = make(map[string][]PortBinding)
	}
	C.PortBindings[port] = append(C.PortBindings[port], PortBinding{"0.0.0.0", port})
}

func (C *Container) AddBind(host string, container string) {
	v := host + ":" + container
	C.Binds = append(C.Binds, v)
	_, found := C.Volumes[container]
	if !found {
		if C.Volumes == nil {
			C.Volumes = make(map[string]string)
		}
		C.Volumes[container] = ""
	}
	return
}

func (C *Container) Start() (err error) {

	log.Printf("Container created id:%s", C.Id)

	bs, err := json.Marshal(C)
	b := bytes.NewBuffer(bs)

	resp, err := C.D.h.Post("containers/"+C.Id+"/start", "application/json", b)
	if err != nil {
		return
	}

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
func (Img *Image) AddTag(D *Docker, tag string) (err error) {
	b := strings.NewReader("")

	tag = url.QueryEscape(tag)
	id := url.QueryEscape(Img.GetName())

	resp, err := D.h.Post("images/"+id+"/tag?repo="+tag+"&force=1", "application/json", b)

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
func (D *Docker) Build(fd io.Reader, name string) (i *Image, err error) {

	v := fmt.Sprintf("%d", time.Now().Unix())
	resp, err := D.h.Post("build?t="+name+"%3A"+v,
		"application/tar", fd)

	if err != nil {
		return
	}

	defer resp.Body.Close()

	i = &Image{}

	LogReader(resp.Body)

	i = NewNamedImage(name + ":" + v)

	err = i.AddTag(D, name)
	return
}

// loadImage pulls a specified image into a docker instance
func (D *Docker) Load(imagename string) (I *Image, err error) {
	b := strings.NewReader("")
	resp, err := D.h.Post("images/create?fromImage="+imagename,
		"text", b)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	LogReader(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("create status code is not 200 %s", resp.StatusCode)
	}

	log.Printf("Image fetched %s", imagename)
	I = NewNamedImage(imagename)
	return
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
