package main

import (
    "encoding/json"
    "io/ioutil"
    "log"
    "os"
)

type Container struct {
    quantity int
    mode string
    granularity string
}

type MachineType struct {
    provider string
    ip []string
}

type SkeletonDeployment struct {
    machines MachineType
    containers []Container
}

func main() {
    config, err := os.Open("bonesFile")

    if err != nil {
        log.Fatal(err)
    }

    configslice, err := ioutil.ReadAll(config)
    if err != nil {
        log.Fatal(err)
    }

    var deploy SkeletonDeployment

    json.Unmarshal(configslice, deploy)

}
