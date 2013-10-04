package main

import (
	"bytes"
	"common"
	"encoding/json"
	"io/ioutil"
	"log"
	"net"
	"os"
	"time"
)

type NoOrchestratorFound struct{}

func (t *NoOrchestratorFound) Error() string {
	return "No Orchestrator Found"
}

// loadBonesFile does initial bones file loading and does some quick and dirty
// data sanitization
func loadBonesFile() *common.SkeletonDeployment {
	log.Print("Loading bonesFile")
	config, err := os.Open("bonesFile")
	if err != nil {
		log.Fatal(err)
	}

	configslice, err := ioutil.ReadAll(config)
	if err != nil {
		log.Fatal(err)
	}

	deploy := new(common.SkeletonDeployment)

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

// findOrchestrator finds a running orchestrator by scanning port 900 on all
// machines it knows about
func findOrchestrator(config *common.SkeletonDeployment) (string, error) {
	log.Print("Finding orchestrator")
	client := common.MakeHttpClient()
	for _, v := range config.Machines.Ip {
		_, err := net.DialTimeout("tcp", v+":900", 1000*time.Millisecond)
		if err != nil {
			continue
		}
		_, err = client.Get("http://" + v + ":900/version")
		if err == nil {
			log.Print("Orchestrator Found")
			return v, nil
		}
	}

	log.Print("No Orchestrator Found")
	return "", new(NoOrchestratorFound)
}

func buildEnv(ip string) []string {
	a := make([]string, 1)
	a[0] = "HOST=" + ip
	return a
}

// bootstrapOrchestrator starts up the orchestrator on a machine
func bootstrapOrchestrator(ip string) string {
	log.Print("Bootstrapping Orchestrator")
	D := common.NewDocker(ip)

	//Setup orchestrator container
	tar := common.TarDir("../../containers/orchestrator")
	err := D.BuildImage(tar, "orchestrator")
	if err != nil {
		log.Fatal(err)
	}

	//Setup gatekeeper container
	tar = common.TarDir("../../containers/gatekeeper")
	err = D.BuildImage(tar, "gatekeeper")
	if err != nil {
		log.Fatal(err)
	}

	//Run orchestrator
	_, err = D.RunImage("orchestrator", buildEnv(D.GetIP()))
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Orchestrator bootstrapped")
	return ip
}

// deploy pushes our new deploy configuration to the orchestrator
func deploy(ip string, config *common.SkeletonDeployment) {

	log.Print("Pushing images to Orchestrator")
	h := common.MakeHttpClient()

	for k, _ := range config.Containers {
		image := common.TarDir(k)
		resp, err := h.Post("http://"+ip+":900/image?name="+k, "application/tar",
			image)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		common.LogReader(resp.Body)
	}

	log.Print("Pushing configuration to Orchestrator")

	barr, err := json.Marshal(config)
	if err != nil {
		log.Fatal(err)
	}

	b := bytes.NewBuffer(barr)

	resp, err := h.Post("http://"+ip+":900/deploy", "application/json", b)
	log.Print("Post returned")

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	log.Print("Deploy Pushed")

	common.LogReader(resp.Body)

	return
}

func main() {

	config := loadBonesFile()

	orch, err := findOrchestrator(config)
	switch err.(type) {

	// Initial Setup
	case *NoOrchestratorFound:
		orch = bootstrapOrchestrator(config.Machines.Ip[0])
		deploy(orch, config)

	// Update Deploy
	case nil:
		D := common.NewDocker(orch)
		D.StopImage("orchestrator")
		D.StopImage("gatekeeper")
		orch = bootstrapOrchestrator(config.Machines.Ip[0])
		deploy(orch, config)

	// Error contacting orchestrator
	default:
		log.Fatal(err)
	}
}
