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
	logger       *log.Logger
	multiplexer  *common.Multiplexer
	imageNames   map[string]string
	key          string
	D            *common.Docker
	c            *libgatekeeper.Client
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
	o.logger.Print("index setup")
	registryName := "samalba/docker-registry"
	o.startImage(registryName, o.repoip, "5000")
}

func (o *orchestrator) StartGatekeeper() {
	o.logger.Print("gatekeeper setup")
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
	o.c.Set("key."+ip+"."+container, container_key)
	o.c.Set("key."+onetime_key, container_key)
	o.c.SwitchOwner("key."+onetime_key, "")
	env[1] = "GATEKEEPER_KEY=" + onetime_key

	return env, nil
}

func (o *orchestrator) startImage(registryName string, portchan chan string, port string) {
	// So that id is passed out of the function
	Img := &common.Image{}

	//To fix loop scoping
	var C *common.Container
	var running bool
	var err error

	for ; ; time.Sleep(10 * time.Second) {
		running, C, err = Img.IsRunning(o.D, registryName)
		if err != nil {
			o.logger.Print(err)
			continue
		}
		if !running {
			o.logger.Print(registryName + " not running")
			Img, err := o.D.Load(registryName)
			if err != nil {
				o.logger.Print(err)
				continue
			}
			C, err = Img.Run(o.D, nil, port)
			if err != nil {
				o.logger.Print(err)
				continue
			}
		}
		break
	}
	o.logger.Print(registryName+" running id: ", C.Id)

	err = C.Inspect()
	if err != nil {
		o.logger.Print(err)
	}
	o.logger.Print(registryName + " fetched config")
	port = C.NetworkSettings.Ports[port+"/tcp"][0]["HostPort"]

	host := o.D.GetIP() + ":" + port

	if err != nil {
		o.logger.Print(err)
	}

	for {
		portchan <- host
	}

}

func (o *orchestrator) handleImage(w http.ResponseWriter, r *http.Request) {
	enc := common.NewEncWriter(w)
	o.multiplexer.Attach(enc)
	defer o.multiplexer.Detach(enc)
	enc.Log("Waiting for index to be downloaded, this may take a while")
	repoip := <-o.repoip
	enc.Log("Recieved\n")

	tag := r.URL.Query()["name"]
	if len(tag) > 0 {
		enc.Log("Building image\n")
		Img, err := o.D.Build(r.Body, tag[0])
		if err != nil {
			enc.SetError(err)
			return
		}
		enc.Log("Tagging\n")
		repo_tag := repoip + "/" + tag[0]
		err = Img.AddTag(o.D, repo_tag)
		if err != nil {
			enc.SetError(err)
			return
		}
		enc.Log("Pushing to index\n")
		err = Img.Push(o.D, enc, repo_tag)
		if err != nil {
			enc.SetError(err)
			return
		}

		o.imageNames[tag[0]] = repo_tag
	}
	enc.Log("built")
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
	enc := common.NewEncWriter(w)
	enc.Log("Starting deploy")
	d := &common.SkeletonDeployment{}
	c, err := ioutil.ReadAll(r.Body)

	if err != nil {
		enc.SetError(err)
		return
	}
	err = json.Unmarshal(c, d)
	if err != nil {
		enc.SetError(err)
		return
	}

	for _, ip := range d.Machines.Ip {
		enc.Log("Adding ip\n" + ip + "\n")
		o.addip <- ip
	}
	enc.Log("Waiting for image refreshes")
	o.WaitRefresh(time.Now())
	enc.Log("waited")

	current := <-o.deploystate

	diff := o.calcUpdate(enc, *d, current)

	sdiff := fmt.Sprint(diff)
	enc.Log("Diff")
	enc.Log(sdiff)

	enc.Log("Deploying diff")
	for ip, images := range diff {
		for _, container := range images {
			D := common.NewDocker(ip)
			Img := &common.Image{}

			enc.Log("Deploying " + container + " on " + ip)
			enc.Log("Indexname " + o.imageNames[container] + "\n")
			Img, err := D.Load(o.imageNames[container])
			if err != nil {
				enc.SetError(err)
				continue
			}

			//Sets environment variables, especially the gatekeeper key
			env, err := o.BuildEnv(ip, container)
			if err != nil {
				enc.SetError(err)
				continue
			}

			expose_port := ""
			if len(d.Containers[container].Expose) > 0 {
				expose_port = d.Containers[container].Expose[0]
			}

			C, err := Img.Run(D, env, expose_port)
			if err != nil {
				enc.SetError(err)
				continue
			}

			enc.Log("Deployed\n" + C.Id + "\n")
		}
	}
}

func NewOrchestrator() (o *orchestrator) {
	o = new(orchestrator)
	o.D = common.NewDocker(os.Getenv("HOST"))
	o.repoip = make(chan string)
	o.gatekeeperip = make(chan string)
	o.multiplexer = common.NewMultiplexer()
	o.logger = log.New(o.multiplexer, "", 0)
	o.imageNames = make(map[string]string)
	o.key = "orchestrator_key"
	go o.StartState()
	go o.StartRepository()
	go o.StartGatekeeper()
	go func() {
		gatekeeperip := <-o.gatekeeperip
		o.c = libgatekeeper.NewClient(gatekeeperip, o.key)
	}()
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

	o.logger.Fatal(http.ListenAndServe(":900", nil))
}
