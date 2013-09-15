package main

import (
	"common"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
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

func (o *orchestrator) WaitRefresh(t time.Time) {
	for {
		s := <-o.deploystate
		good := true
		for _, v := range s {
			if v.Updated.After(t) {
				good = false
				break

			}
		}
		if good {
			break
		}
		time.Sleep(10 * time.Second)
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

func (o *orchestrator) calcUpdate(desired common.SkeletonDeployment, current map[string]common.DockerInfo) (update map[string][]string) {
	// Maps IP's to lists of containers to deploy
	update = make(map[string][]string)
	// For each container we want to deploy
	for container, _ := range desired.Containers {
		// Assuming granularity machine

		// For each machine check for container
		for ip, mInfo := range current {

			//Have we found the container
			found := false

			//Check if the container is running
			for _, checkContainer := range mInfo.Containers {

				imageName := checkContainer.Id

				//Get the actual name
				if strings.Contains(imageName, "/") {
					imageName = strings.SplitN(imageName, "/", 2)[1]
				}
				if strings.Contains(imageName, ":") {
					imageName = strings.SplitN(imageName, ":", 2)[0]
				}
				if imageName == container {
					found = true
					break
				}
			}

			//Do we need to deploy a image?
			if !found {
				update[ip] = append(update[ip], container)
			}

		}
	}

	return update

}

func (o *orchestrator) deploy(w http.ResponseWriter, r *http.Request) {

	io.WriteString(w, "Starting deploy\n")
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

	io.WriteString(w, "Waiting for image refreshes\n")
	o.WaitRefresh(time.Now())
	io.WriteString(w, "waited\n")

	current := <-o.deploystate

	diff := o.calcUpdate(*d, current)

	sdiff := fmt.Sprint(diff)
	io.WriteString(w, sdiff)
	io.WriteString(w, "\n")

	indexip := <-o.repoip

	io.WriteString(w, "Deploying diff\n")
	for ip, images := range diff {
		for _, container := range images {
			io.WriteString(w, "Deploying "+container+" on "+ip+"\n")
			err := common.LoadImage(ip, indexip+"/"+container)
			if err != nil {
				io.WriteString(w, err.Error())
				continue
			}
			id, err := common.RunImage(ip, indexip+"/"+container, false)
			io.WriteString(w, "Deployed \n")
			io.WriteString(w, id)
			io.WriteString(w, "\n")
			if err != nil {
				io.WriteString(w, err.Error())
			}
			io.WriteString(w, "\n")
		}
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
