package main

import (
	"code.google.com/p/go.crypto/ssh"
	"encoding/json"
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
	log.Print("parsing configuration file")
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

	log.Print("Configuration parsed")
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

func bootstrapOrchestrator(ip string) string {
	c, err := net.DialTimeout("tcp", ip+":22", 100*time.Millisecond)
	if err != nil {
		log.Fatal(err)
	}
	_, _ = ssh.Client(c, nil)
	return ""

}

func deploy(orchestratorip string) {
	return

}

func main() {

	config := loadBonesFile()
	log.Print("bonesFile Parsed")

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
