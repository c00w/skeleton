package main

import (
	"common"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type orchestrator struct {
	repoip chan string
}

func (o *orchestrator) StartRepository() {
	log.Print("index setup")
	registry_name := "samalba/docker-registry"
	host := os.Getenv("HOST")
	for ; ; time.Sleep(10 * time.Second) {
		running, _, err := common.ImageRunning(host, registry_name)
		if err != nil {
			log.Print(err)
			continue
		}
		if !running {
			log.Print("index not running")
			err := common.LoadImage(host, registry_name)
			if err != nil {
				log.Print(err)
				continue
			}
			err = common.RunImage(host, registry_name, false)
			if err != nil {
				log.Print(err)
				continue
			}
		}
		break
	}
	log.Print("index running")
	for {
		o.repoip <- host
	}

}

func (o *orchestrator) handleImage(w http.ResponseWriter, r *http.Request) {
	_ = <-o.repoip
	io.WriteString(w, "Recieved\n")
	tag := r.URL.Query()["name"]
	if len(tag) > 0 {
		io.WriteString(w, "Building image\n")
		err := common.BuildImage(os.Getenv("HOST"), r.Body, tag[0])
		if err != nil {
			io.WriteString(w, err.Error()+"\n")
		}
	}
	io.WriteString(w, "built\n")
}

func main() {

	o := new(orchestrator)
	o.repoip = make(chan string)

	go o.StartRepository()

	http.HandleFunc("/version", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "v0")
	})

	http.HandleFunc("/image", o.handleImage)

	http.HandleFunc("/deploy", func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "deployed")
	})

	log.Fatal(http.ListenAndServe(":900", nil))
}
