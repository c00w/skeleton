package main

import (
	"common"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net"
	"os"
	"strings"
	"time"
	"crypto/tls"
)

type orchestrator struct {
	repoip       chan string
	gatekeeperip chan string
	deploystate  chan map[string]common.DockerInfo
	addip        chan string
	logger       *log.Logger
	multiplexer  *common.Multiplexer
	D            *common.Docker
}

func (o *orchestrator) pollDocker(ip string, update chan common.DockerInfo) {
	for ; ; time.Sleep(60 * time.Second) {
		c, err := o.D.ListContainers()
		if err != nil {
			o.logger.Print(err)
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
	o.logger.Print("index setup")
	registryName := "samalba/docker-registry"
	o.startImage(registryName, o.repoip, "5000")
}

func (o *orchestrator) StartGatekeeper() {
	o.logger.Print("gatekeeper setup")
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
			o.logger.Print(err)
			continue
		}
		if !running {
			log.Print(registryName + " not running")
			err := o.D.LoadImage(registryName)
			if err != nil {
				o.logger.Print(err)
				continue
			}
			id, err = o.D.RunImage(registryName, nil)
			if err != nil {
				o.logger.Print(err)
				continue
			}
		}
		break
	}
	o.logger.Print(registryName+" running id: ", id)
	config, err := o.D.InspectContainer(id)
	o.logger.Print(registryName + "fetched config")
	port = config.NetworkSettings.PortMapping.Tcp[port]

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
		err := o.D.BuildImage(r.Body, tag[0])
		if err != nil {
			enc.SetError(err)
			return
		}
		enc.Log("Tagging\n")
		repo_tag := repoip + "/" + tag[0]
		err = o.D.TagImage(tag[0], repo_tag)
		if err != nil {
			enc.SetError(err)
			return
		}
		enc.Log("Pushing to index\n")
		err = o.D.PushImage(w, repo_tag)
		if err != nil {
			enc.SetError(err)
			return
		}
	}
	enc.Log("built")
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

	diff := o.calcUpdate(w, *d, current)

	sdiff := fmt.Sprint(diff)
	enc.Log(sdiff)

	indexip := <-o.repoip
	enc.Log("Deploying diff")
	for ip, images := range diff {
		for _, container := range images {
			D := common.NewDocker(ip)
			enc.Log("Deploying " + container + " on " + ip)
			err := D.LoadImage(indexip + "/" + container)
			if err != nil {
				enc.SetError(err)
				continue
			}
			id, err := D.RunImage(indexip+"/"+container, o.BuildEnv())
			enc.Log("Deployed\n"+id+"\n")
			if err != nil {
				enc.SetError(err)
			}
		}
	}
}

func NewOrchestrator() (o *orchestrator) {
	o = new(orchestrator)
	o.repoip = make(chan string)
	o.gatekeeperip = make(chan string)
	o.multiplexer = common.NewMultiplexer()
	o.logger = log.New(o.multiplexer, "", 0)
	go o.StartState()
	go o.StartRepository()
	go o.StartGatekeeper()
	o.D = common.NewDocker(os.Getenv("HOST"))
	return o
}

func status(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "status page")
}

func listenAndServeOrchestratorTLS(d *http.ServeMux) error {
    cert,key := common.GenerateCertificate(os.Getenv("Host"))
    server := &http.Server{Addr: ":900", Handler: d}
    config := &tls.Config{}
    *config = *server.TLSConfig
    if config.NextProtos == nil {
        config.NextProtos = []string{"http/1.1"}
    }
    var err error
    config.Certificates = make([]tls.Certificate, 1)
    config.Certificates[0], err = tls.X509KeyPair(cert,key)
    if err != nil {
        return err
       }
    conn, err := net.Listen("tcp",server.Addr)
    if err != nil {
        return err
    }
    
    tlsListener := tls.NewListener(conn,config)
    return server.Serve(tlsListener)
}
func main() {

	o := NewOrchestrator()
        
	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "orchestrator v0")
	})

	http.HandleFunc("/image", o.handleImage)

	http.HandleFunc("/deploy", o.deploy)
        
	o.logger.Fatal(listenAndServeOrchestratorTLS(http.DefaultServeMux))
}
