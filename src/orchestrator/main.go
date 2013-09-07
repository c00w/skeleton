package main

import (
	"common"
	"io"
	"log"
	"net/http"
	"os"
)

type orchestrator struct {
	repoip chan string
}

func (o *orchestrator) StartRepository() {
	registry_name := "samalba/docker-registry"
	host := os.Getenv("HOST")
	running, _, err := common.ImageRunning(host, registry_name)
	if err != nil {
		log.Fatal(err)
	}
	if !running {
		err := common.LoadImage(host, registry_name)
		err = common.RunImage(host, registry_name, false)
		if err != nil {
			log.Print(err)
		}
	}

	for {
		o.repoip <- host
	}

}

func (o *orchestrator) handleImage(w http.ResponseWriter, r *http.Request) {
	return
	tag := r.URL.Query()["name"]
	if len(tag) > 0 {
		common.BuildImage(os.Getenv("HOST"), r.Body, tag[0])
	}
	io.WriteString(w, "built")
}

func main() {

	o := new(orchestrator)
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
