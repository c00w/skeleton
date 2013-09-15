package main

import (
	"common"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

type orchestrator struct {
	repoip      chan string
	deploystate chan map[string]common.DockerInfo
	addip       chan string
}

func (o *orchestrator) pollDocker(ip string, update chan common.DockerInfo) {
	for ; ; time.Sleep(60 * time.Second) {
		c, err := common.ListContainers(ip)
		if err != nil {
			log.Print(err)
			continue
		}
		d := common.DockerInfo{ip, c, nil, time.Now()}
		update <- d
	}
}

func (o *orchestrator) StartState() {
	d := make(map[string]common.DockerInfo)
	o.deploystate = make(chan map[string]common.DockerInfo)
	o.addip = make(chan string)
	updatechan := make(chan common.DockerInfo)
	for {
		select {
		case o.deploystate <- d:

		case ip := <-o.addip:
			_, exist := d[ip]
			if !exist {
				d[ip] = common.DockerInfo{}
			}
			go o.pollDocker(ip, updatechan)

		case up := <-updatechan:
			d[up.Ip] = up
		}
	}
}

func (o *orchestrator) StartRepository() {
	log.Print("index setup")
	registry_name := "samalba/docker-registry"
	host := os.Getenv("HOST")
	// So that id is passed out of the function
	id := ""
	var err error
	var running bool
	for ; ; time.Sleep(10 * time.Second) {
		running, id, err = common.ImageRunning(host, registry_name)
		if err != nil {
			log.Print(err)
			continue
		}
		if !running {
			log.Print("index not running")
			err := common.LoadImage(host, registry_name)
			if err != nil {
				log.Print(err)
				continue
			}
			id, err = common.RunImage(host, registry_name, false)
			if err != nil {
				log.Print(err)
				continue
			}
		}
		break
	}
	log.Print("index running id: ", id)
	config, err := common.InspectContainer(host, id)
	log.Print("fetched config")
	ip := config.NetworkSettings.PortMapping.Tcp["5000"]

	host = host + ":" + ip

	if err != nil {
		log.Print(err)
	}

	o.repoip = make(chan string)
	for {
		o.repoip <- host
	}

}

func (o *orchestrator) handleImage(w http.ResponseWriter, r *http.Request) {
	repoip := <-o.repoip
	io.WriteString(w, "Recieved\n")
	tag := r.URL.Query()["name"]
	if len(tag) > 0 {
		io.WriteString(w, "Building image\n")
		err := common.BuildImage(os.Getenv("HOST"), r.Body, tag[0])
		if err != nil {
			io.WriteString(w, err.Error()+"\n")
			return
		}

		io.WriteString(w, "Tagging\n")
		repo_tag := repoip + "/" + tag[0]
		err = common.TagImage(os.Getenv("HOST"), tag[0], repo_tag)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		io.WriteString(w, "Pushing to index\n")
		err = common.PushImage(os.Getenv("HOST"), w, repo_tag)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
	}
	io.WriteString(w, "built\n")
}

func (o *orchestrator) deploy(w http.ResponseWriter, r *http.Request) {

	io.WriteString("Starting deploy")
	d := &common.SkeletonDeployment{}
	c, err := ioutil.ReadAll(r.Body)

	if err != nil {
		io.WriteString(w, err.Error())
		return
	}
	err = json.Unmarshal(c, d)
	if err != nil {
		io.WriteString(w, err.Error())
		return
	}

	for _, ip := range d.Machines.Ip {
		io.WriteString(w, "Adding ip\n")
		io.WriteString(w, ip)
		o.addip <- ip
	}
}

func main() {

	o := new(orchestrator)

	go o.StartRepository()
	go o.StartState()

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "v0")
	})

	http.HandleFunc("/image", o.handleImage)

	http.HandleFunc("/deploy", o.deploy)

	log.Fatal(http.ListenAndServe(":900", nil))
}
