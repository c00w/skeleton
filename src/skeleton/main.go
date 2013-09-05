package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
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

func logReader(r io.Reader) {
	buff := make([]byte, 1024)
	for _, err := r.Read(buff); err == nil; _, err = r.Read(buff) {
		log.Print(string(buff))
	}
}

type idDump struct {
	Id string
}

// runContainer takes a ip and a docker image to run, and makes sure it is running
func runContainer(ip string, image string) {
	h := makeHttpClient()
	b := bytes.NewBuffer([]byte("{\"Image\":\"" + image + "\"}"))
	resp, err := h.Post("http://"+ip+":4243/containers/create",
		"application/json", b)
	defer resp.Body.Close()

	s, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 201 {
		log.Fatal("response status code not 201")
	}

	if err != nil {
		log.Fatal(err)
	}

	a := new(idDump)
	json.Unmarshal(s, a)
	id := a.Id

	log.Printf("Container created id:%s", id)

	b = bytes.NewBuffer([]byte("{}"))
	resp, err = h.Post("http://"+ip+":4243/containers/"+id+"/start",
		"application/json", b)
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		logReader(resp.Body)
		log.Fatal("start status code is not 204 it is %i", resp.StatusCode)

	}

	log.Printf("Container running")

}

// buildImage takes a tarpath, and builds it
func buildImage(ip string, tarpath string, name string) {

	fd := tarDir(tarpath)

	h := makeHttpClient()
	resp, err := h.Post("http://"+ip+":4243/build?t="+name,
		"application/tar", fd)

	if err != nil {
		log.Fatal(err)
	}

	defer resp.Body.Close()

	logReader(resp.Body)
	log.Print(resp.StatusCode)
}

// loadImage pulls a specified image into a docker instance
func loadImage(ip string, image string) {
	h := makeHttpClient()
	b := bytes.NewBuffer(nil)
	resp, err := h.Post("http://"+ip+":4243/images/create?fromImage="+image,
		"text", b)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	logReader(resp.Body)
	if resp.StatusCode != 200 {
		log.Printf("create status code is not 200 %s", resp.StatusCode)
	}

	log.Printf("Image fetched %s", image)
}

// bootstrapOrchestrator starts up the orchestrator on a machine
func bootstrapOrchestrator(ip string) string {
	log.Print("Bootstrapping Orchestrator")
	buildImage(ip, "../../containers/orchestrator", "orchestrator")
	runContainer(ip, "orchestrator")
	log.Print("Orchestrator bootstrapped")
	return ip
}

// deploy pushes our new deploy configuration to the orchestrator
func deploy(ip string, config *SkeletonDeployment) {

	log.Print("Pushing configuration to Orchestrator")

	h := makeHttpClient()
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

	logReader(resp.Body)

	return
}

// tarDir takes a directory path and produces a reader which is all of its
// contents tarred up and compressed with gzip
func tarDir(path string) io.Reader {

	// check this is a directory
	log.Print(path)
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
		log.Print(h.Name)

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
		deploy(orch, config)

	// Error contacting orchestrator
	default:
		log.Fatal(err)
	}
}
