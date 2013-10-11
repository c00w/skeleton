package main

import (
	"common"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"libgatekeeper"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type orchestrator struct {
	repoip       chan string
	gatekeeperip chan string
	deploystate  chan map[string]*common.Docker
	addip        chan string
	imageNames   map[string]string
	key          string
	D            *common.Docker
}

func (o *orchestrator) StartState() {
	d := make(map[string]*common.Docker)
	o.deploystate = make(chan map[string]*common.Docker)
	o.addip = make(chan string)
	for {
		select {
		case o.deploystate <- d:

		case ip := <-o.addip:
			_, exist := d[ip]
			if !exist {
				d[ip] = common.NewDocker(ip)
			}
			go d[ip].Update()
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

func (o *orchestrator) BuildEnv(ip string, container string) ([]string, error) {
	gid := <-o.gatekeeperip
	env := make([]string, 2)
	env[0] = "GATEKEEPER=" + gid

	//Create container key
	b := make([]byte, 64)
	for n := 0; n < 64; {
		t, err := rand.Read(b)
		if err != nil {
			return nil, err
		}
		n += t
	}
	container_key := hex.EncodeToString(b[0:32])
	onetime_key := hex.EncodeToString(b[0:32])
	gatekeeperip := <-o.gatekeeperip
	c := libgatekeeper.NewClient(gatekeeperip, o.key)
	c.Set("key."+ip+"."+container, container_key)
	c.Set("key."+onetime_key, container_key)
	c.SwitchOwner("key."+onetime_key, "")
	env[1] = "GATEKEEPER_KEY=" + onetime_key

	return env, nil
}

func (o *orchestrator) startImage(registryName string, portchan chan string, port string) {
	// So that id is passed out of the function
	id := ""
	Img := &common.Image{}
	for ; ; time.Sleep(10 * time.Second) {
		running, _, err := Img.IsRunning(o.D, registryName)
		if err != nil {
			log.Print(err)
			continue
		}
		if !running {
			log.Print(registryName + " not running")
			err := Img.Load(o.D, registryName)
			if err != nil {
				log.Print(err)
				continue
			}
			id, err = Img.Run(o.D, registryName, nil)
			if err != nil {
				log.Print(err)
				continue
			}
		}
		break
	}

	log.Print(registryName+" running id: ", id)
	C := &common.Container{}
	C.Id = id
	C.D = o.D
	err := C.Inspect()
	log.Print(registryName + "fetched config")
	port = C.NetworkSettings.PortMapping.Tcp[port]

	host := o.D.GetIP() + ":" + port

	if err != nil {
		log.Print(err)
	}

	for {
		portchan <- host
	}

}

func (o *orchestrator) handleImage(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Waiting for index to be downloaded, this may take a while\n")
	repoip := <-o.repoip
	io.WriteString(w, "Recieved\n")

	Img := &common.Image{}

	tag := r.URL.Query()["name"]
	if len(tag) > 0 {
		io.WriteString(w, "Building image\n")
		err := Img.Build(o.D, r.Body, tag[0])
		if err != nil {
			io.WriteString(w, err.Error()+"\n")
			return
		}

		io.WriteString(w, "Tagging\n")
		repo_tag := repoip + "/" + tag[0]
		err = Img.AddTag(o.D, tag[0], repo_tag)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}
		io.WriteString(w, "Pushing to index\n")
		err = Img.Push(o.D, w, repo_tag)
		if err != nil {
			io.WriteString(w, err.Error())
			return
		}

		o.imageNames[tag[0]] = repo_tag
	}
	io.WriteString(w, "built\n")
}

func (o *orchestrator) calcUpdate(w io.Writer, desired common.SkeletonDeployment, current map[string]*common.Docker) (update map[string][]string) {
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
		io.WriteString(w, "\n")
		o.addip <- ip
	}

	io.WriteString(w, "Waiting for image refreshes\n")
	o.WaitRefresh(time.Now())
	io.WriteString(w, "waited\n")

	current := <-o.deploystate

	diff := o.calcUpdate(w, *d, current)

	sdiff := fmt.Sprint(diff)
	io.WriteString(w, sdiff)
	io.WriteString(w, "\n")

	io.WriteString(w, "Deploying diff\n")
	for ip, images := range diff {
		for _, container := range images {
			D := common.NewDocker(ip)
			Img := &common.Image{}

			io.WriteString(w, "Deploying "+container+" on "+ip+"\n")
			io.WriteString(w, "Indexname "+o.imageNames[container]+"\n")
			err := Img.Load(D, o.imageNames[container])
			if err != nil {
				io.WriteString(w, err.Error())
				io.WriteString(w, "\n")
				continue
			}

			env, err := o.BuildEnv(ip, container)
			if err != nil {
				io.WriteString(w, err.Error())
				io.WriteString(w, "\n")
				continue
			}

			id, err := Img.Run(D, o.imageNames[container], env)
			if err != nil {
				io.WriteString(w, err.Error())
				io.WriteString(w, "\n")
				continue
			}

			io.WriteString(w, "Deployed \n")
			io.WriteString(w, id)
			io.WriteString(w, "\n")
		}
	}
}

func NewOrchestrator() (o *orchestrator) {
	o = new(orchestrator)
	o.repoip = make(chan string)
	o.gatekeeperip = make(chan string)
	o.D = common.NewDocker(os.Getenv("HOST"))
	o.imageNames = make(map[string]string)
	o.key = "orchestrator_key"
	go o.StartState()
	go o.StartRepository()
	go o.StartGatekeeper()
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
