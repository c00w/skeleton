package main

import (
	"archive/tar"
	"bytes"
	"common"
	"compress/gzip"
	"encoding/json"
	"io"
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

// bootstrapOrchestrator starts up the orchestrator on a machine
func bootstrapOrchestrator(ip string) string {
	log.Print("Bootstrapping Orchestrator")
	tar := tarDir("../../containers/orchestrator")
	common.BuildImage(ip, tar, "orchestrator")
	err := common.RunImage(ip, "orchestrator", true)
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
		image := tarDir(k)
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

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	log.Print("Deploy Pushed")

	common.LogReader(resp.Body)

	return
}

// tarDir takes a directory path and produces a reader which is all of its
// contents tarred up and compressed with gzip
func tarDir(path string) io.Reader {
	log.Print("compressing ", path)
	// check this is a directory
	i, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	if !i.IsDir() {
		log.Fatal("Directory to tar up is not a directory")
	}

	//Make a buffer to hold the file
	b := bytes.NewBuffer(nil)
	g := gzip.NewWriter(b)
	w := tar.NewWriter(g)

	// Find subdirectories
	ifd, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}
	fi, err := ifd.Readdir(-1)
	if err != nil {
		log.Fatal(err)
	}

	// Put them in tarfile
	for _, f := range fi {
		h, err := tar.FileInfoHeader(f, "")
		if err != nil {
			log.Fatal(err)
		}

		w.WriteHeader(h)

		ffd, err := os.Open(path + "/" + f.Name())

		if err != nil {
			log.Fatal(err)
		}

		c, err := ioutil.ReadAll(ffd)
		if err != nil {
			log.Fatal(err)
		}
		w.Write(c)
	}
	w.Close()
	g.Close()
	return b
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
		common.StopImage(orch, "orchestrator")
		orch = bootstrapOrchestrator(config.Machines.Ip[0])
		deploy(orch, config)

	// Error contacting orchestrator
	default:
		log.Fatal(err)
	}
}
