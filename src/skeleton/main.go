package main

import (
	"bytes"
	"common"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"time"
	"errors"
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

		//Granularity defaults to deployment
		if len(v.Granularity) == 0 {
			v.Granularity = "deployment"
		}

		// Source defaults to local:<name>
		if len(v.Source) == 0 {
			v.Source = "local:" + k
		}

		// Set mode to default
		if len(v.Mode) == 0 {
			v.Mode = "default"
		}
		deploy.Containers[k] = v
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
	Img := &common.Image{}

	//Setup gatekeeper image
	tar := common.TarDir("../../containers/gatekeeper")
	Img, err := D.Build(tar, "gatekeeper")
	if err != nil {
		log.Fatal(err)
	}

	//Setup orchestrator container
	tar = common.TarDir("../../containers/orchestrator")

	Img, err = D.Build(tar, "orchestrator")
	if err != nil {
		log.Fatal(err)
	}
	_, err = Img.Run(D, buildEnv(D.GetIP()))
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Orchestrator bootstrapped")
	return ip
}

// deploys the images to the server
func deployImages(ip string, config *common.SkeletonDeployment) error {
       log.Print("Pushing images to Orchestrator")
        h := common.MakeHttpClient()
        var image io.Reader
        for k, v := range config.Containers {
                source := strings.SplitN(v.Source, ":", 2)
                if source[0] == "local" {
                        image = common.TarDir(source[1])
                } else {
                        log.Fatal(source)
                }
                resp, err := h.Post("http://"+ip+":900/image?name="+k, "application/tar",
                        image)
                if err != nil {
                        log.Fatal(err)
                }
                defer resp.Body.Close()
                err = common.JsonReader(resp.Body)
                if err != nil {
                        return err
                }
        }
        return nil 
}
// dpeloys the configuration to the server
func deployConfig(ip string, config *common.SkeletonDeployment) error {
        h := common.MakeHttpClient()
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

        err = nil
        err = common.JsonReader(resp.Body)

        return err        
}

// deploy pushes our new deploy configuration to the orchestrator
func deploy(ip string, config *common.SkeletonDeployment, update string) error {
        err := errors.New("Incorrect command entered")
        if update == "deploy"{
                deployImages(ip, config)
                err = deployConfig(ip, config)
        } else if update == "push images"{
                err = deployImages(ip, config)
        } else if update == "push configuration"{
                err = deployConfig(ip, config)
        }
        log.Print("Deploy Pushed")
        return err
}

func main() {

        flag.Parse()
        if flag.NArg() > 1 || (len(flag.Args())==0) {
                log.Print("Error - bring up help flags")
        }else if flag.Arg(0) == "v" || flag.Arg(0) == "version"{
                log.Print("prints version number")
        }else{
        	config := loadBonesFile()

        	orch, err := findOrchestrator(config)
        	switch err.(type) {

        	// Initial Setup
        	case *NoOrchestratorFound:
        		orch = bootstrapOrchestrator(config.Machines.Ip[0])
        		err = deploy(orch, config, flag.Arg(0))
        		if err != nil {
        			log.Fatal(err)
        		}

        	// Update Deploy
        	case nil:
        		D := common.NewDocker(orch)
        		Img := &common.Image{}
        		Img.Stop(D, "orchestrator")
        		Img.Stop(D, "gatekeeper")
        		orch = bootstrapOrchestrator(config.Machines.Ip[0])
        		err = deploy(orch, config, flag.Arg(0))
        		if err != nil {
        			log.Fatal(err)
        		}

        	// Error contacting orchestrator
        	default:
        		log.Fatal(err)
        	}
	}
}
