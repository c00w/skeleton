package main

import (
    "encoding/json"
    "io/ioutil"
    "log"
    "os"
)

type Container struct {
    Quantity int
    Mode string
    Granularity string
}

type MachineType struct {
    Provider string
    Ip []string
}

type SkeletonDeployment struct {
    Test string
    Machines MachineType
    Containers map[string]Container
}

func loadBonesFile() *SkeletonDeployment {

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

    for k,v := range deploy.Containers {
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

    return deploy
}

func main() {
    _ = loadBonesFile()
    log.Print("bonesFile Parsed")
}
