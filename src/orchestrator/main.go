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
	repoip       chan string
	gatekeeperip chan string
	deploystate  chan map[string]common.DockerInfo
	addip        chan string
	D            *common.Docker
}

func (o *orchestrator) pollDocker(ip string, update chan common.DockerInfo) {
	for ; ; time.Sleep(60 * time.Second) {
		c, err := o.D.ListContainers()
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
		good := false
		for _, v := range s {
			if v.Updated.After(t) {
				good = true
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
	registryName := "samalba/docker-registry"
	o.startImage(registryName, o.repoip, "5000")
}

func (o *orchestrator) StartGatekeeper() {
	log.Print("gatekeeper setup")
	registryName := "gatekeeper"
	o.startImage(registryName, o.gatekeeperip, "800")
}

func (o *orchestrator) BuildEnv() []string {
	gid := <-o.gatekeeperip
	env := make([]string, 1)
	env[0] = "GATEKEEPER=" + gid
	return env
}

func (o *orchestrator) startImage(registryName string, portchan chan string, port string) {
	// So that id is passed out of the function
	id := ""
	var err error
	var running bool
	for ; ; time.Sleep(10 * time.Second) {
		running, id, err = o.D.ImageRunning(registryName)
		if err != nil {
			log.Print(err)
			continue
		}
		if !running {
			log.Print(registryName + " not running")
			err := o.D.LoadImage(registryName)
			if err != nil {
				log.Print(err)
				continue
			}
			id, err = o.D.RunImage(registryName, nil)
			if err != nil {
				log.Print(err)
				continue
			}
		}
		break
	}
	log.Print(registryName+" running id: ", id)
	config, err := o.D.InspectContainer(id)
	log.Print(registryName + "fetched config")
	port = config.NetworkSettings.PortMapping.Tcp[port]

	host := o.D.GetIP() + ":" + port

	if err != nil {
		log.Print(err)
	}

	for {
		portchan <- host
	}

}

func (o *orchestrator) handleImage(w http.ResponseWriter, r *http.Request) {
	enc := common.NewEncWriter(w)
	enc.Write("Waiting for index to be downloaded, this may take a while")
	repoip := <-o.repoip
	enc.Write("Recieved\n")
	tag := r.URL.Query()["name"]
	if len(tag) > 0 {
		enc.Write("Building image\n")
		err := o.D.BuildImage(r.Body, tag[0])
		if err != nil {
			enc.ErrWrite(err)
			return
		}
		enc.Write("Tagging\n")
		repo_tag := repoip + "/" + tag[0]
		err = o.D.TagImage(tag[0], repo_tag)
		if err != nil {
			enc.ErrWrite(err)
			return
		}
		enc.Write("Pushing to index\n")
		err = o.D.PushImage(w, repo_tag)
		if err != nil {
			enc.ErrWrite(err)
			return
		}
	}
	enc.Write("built")
}

func (o *orchestrator) calcUpdate(w io.Writer, desired common.SkeletonDeployment, current map[string]common.DockerInfo) (update map[string][]string) {
	c := fmt.Sprint(current)
	io.WriteString(w, c)
	io.WriteString(w, "\n")
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

				imageName := checkContainer.Image

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
	enc := common.NewEncWriter(w)
	enc.Write("Starting deploy")
	d := &common.SkeletonDeployment{}
	c, err := ioutil.ReadAll(r.Body)

	if err != nil {
		enc.ErrWrite(err)
		return
	}
	err = json.Unmarshal(c, d)
	if err != nil {
		enc.ErrWrite(err)
		return
	}

	for _, ip := range d.Machines.Ip {
		enc.Write("Adding ip\n" + ip + "\n")
		o.addip <- ip
	}
	enc.Write("Waiting for image refreshes")
	o.WaitRefresh(time.Now())
	enc.Write("waited")

	current := <-o.deploystate

	diff := o.calcUpdate(w, *d, current)

	sdiff := fmt.Sprint(diff)
	enc.Write(sdiff)

	indexip := <-o.repoip
	enc.Write("Deploying diff")
	for ip, images := range diff {
		for _, container := range images {
			D := common.NewDocker(ip)
			enc.Write("Deploying " + container + " on " + ip)
			err := D.LoadImage(indexip + "/" + container)
			if err != nil {
				enc.ErrWrite(err)
				continue
			}
			id, err := D.RunImage(indexip+"/"+container, o.BuildEnv())
			enc.Write("Deployed\n"+id+"\n")
			if err != nil {
				enc.ErrWrite(err)
			}
		}
	}
}

func NewOrchestrator() (o *orchestrator) {
	o = new(orchestrator)
	o.repoip = make(chan string)
	o.gatekeeperip = make(chan string)
	go o.StartState()
	go o.StartRepository()
	go o.StartGatekeeper()
	o.D = common.NewDocker(os.Getenv("HOST"))
	return o
}

func status(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "status page")
}

func main() {

	o := NewOrchestrator()

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "orchestrator v0")
	})

	http.HandleFunc("/image", o.handleImage)

	http.HandleFunc("/deploy", o.deploy)

	log.Fatal(http.ListenAndServe(":900", nil))
}
