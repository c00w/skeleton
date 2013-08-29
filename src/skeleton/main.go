package main

import (
	"bytes"
	"code.google.com/p/go.crypto/ssh"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type Container struct {
	Quantity    int
	Mode        string
	Granularity string
}

type MachineType struct {
	Provider string
	Ip       []string
}

type SkeletonDeployment struct {
	Test       string
	Machines   MachineType
	Containers map[string]Container
}

type NoOrchestratorFound struct{}

func (t *NoOrchestratorFound) Error() string {
	return "No Orchestrator Found"
}

// loadBonesFile does initial bones file loading and does some quick and dirty
// data sanitization
func loadBonesFile() *SkeletonDeployment {
	log.Print("Loading bonesFile")
	config, err := os.Open("bonesFile")
	if err != nil {
		log.Fatal(err)
	}

	configslice, err := ioutil.ReadAll(config)
	if err != nil {
		log.Fatal(err)
	}

	deploy := new(SkeletonDeployment)

	err = json.Unmarshal(configslice, deploy)
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range deploy.Containers {
		if len(v.Granularity) == 0 {
			v.Granularity = "deployment"
			deploy.Containers[k] = v
		}
		if len(v.Mode) == 0 {
			v.Mode = "default"
			deploy.Containers[k] = v
		}
	}

	if len(deploy.Machines.Provider) == 0 {
		log.Fatal("Machine Provider must be specified")
	}

	log.Print("bonesFile loaded")
	return deploy
}

func makeHttpClient() *http.Client {
	return &http.Client{}
}

// findOrchestrator finds a running orchestrator by scanning port 900 on all
// machines it knows about
func findOrchestrator(config *SkeletonDeployment) (string, error) {
	log.Print("Finding orchestrator")
	client := makeHttpClient()
	for _, v := range config.Machines.Ip {
		_, err := net.DialTimeout("tcp", "v"+":900", 100*time.Millisecond)
		if err != nil {
			continue
		}
		_, err = client.Get("http://" + v + ":900:/version")
		if err == nil {
			log.Print("Orchestrator Found")
			return v, nil
		}
	}

	log.Print("No Orchestrator Found")
	return "", new(NoOrchestratorFound)
}

// passwordShell is a quick and dirty way to prompt a user for a password
type passwordShell struct{}

func (p *passwordShell) Password(user string) (password string, err error) {

	fmt.Printf("Password: ")
	_, err = fmt.Scanln(&password)
	return password, err
}

// sshClient is a quick and dirty function to get a ssh client to a ip address
func sshClient(ip string) *ssh.ClientConn {
	fmt.Printf("Username: ")
	var username string
	_, err := fmt.Scanln(&username)
	if err != nil {
		log.Fatal(err)
	}

	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.ClientAuth{
			ssh.ClientAuthPassword(&passwordShell{}),
		},
	}
	c, err := ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		log.Fatal(err)
	}
	return c
}

func loopPrint(b *bytes.Buffer, c chan int) {
	for {
		timeout := time.After(50 * time.Millisecond)
		select {
		case _ = <-timeout:
			timeout = time.After(50 * time.Millisecond)
			for {
				l, _ := b.ReadString('\n')
				if len(l) == 0 {
					break
				}
				log.Print(l)
			}
		case _ = <-c:
			return
		}
	}
}

// setupRegistry sets up a locally hosted docker registry on a machine
// mainly intended for bootstrapping the orchestrator
func setupRegistry(ip string) {
	log.Printf("Setting up registry on %s", ip)

	c := sshClient(ip)

	s, err := c.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	log.Print("Connection Opened")

	var b1 bytes.Buffer
	var b2 bytes.Buffer
	cq := make(chan int)
	s.Stdout = &b1
	s.Stderr = &b2
	go loopPrint(&b1, cq)
	go loopPrint(&b2, cq)
	err = s.Run("/usr/bin/sudo /usr/bin/docker run -d samalba/docker-registry")
	time.Sleep(60)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("registry setup")
}

// bootstrapOrchestrator starts up the orchestrator on a machine
func bootstrapOrchestrator(ip string) string {
	log.Print("Bootstrapping Orchestrator")
	_ = sshClient(ip)
	return ""

}

// deploy pushes our new deploy configuration to the orchestrator
func deploy(orchestratorip string) {
	return
}

func main() {

	config := loadBonesFile()

	orch, err := findOrchestrator(config)
	switch err.(type) {

	// Initial Setup
	case *NoOrchestratorFound:
		setupRegistry(config.Machines.Ip[0])
		orch = bootstrapOrchestrator(config.Machines.Ip[0])
		deploy(orch)

	// Update Deploy
	case nil:
		deploy(orch)

	// Error contacting orchestrator
	default:
		log.Fatal(err)
	}
}
