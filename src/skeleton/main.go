package main

import (
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

type passwordShell struct{}

func (p *passwordShell) Password(user string) (password string, err error) {

	fmt.Printf("Password: ")
	_, err = fmt.Scanln(&password)
	return password, err
}

func bootstrapOrchestrator(ip string) string {
	log.Print("Bootstrapping Orchestrator")

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
	_, err = ssh.Dial("tcp", ip+":22", config)
	if err != nil {
		log.Fatal(err)
	}
	return ""

}

func deploy(orchestratorip string) {
	return
}

func main() {

	config := loadBonesFile()

	orch, err := findOrchestrator(config)
	switch err.(type) {

	// Initial Setup
	case *NoOrchestratorFound:
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
