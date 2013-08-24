package main

import (
    "fmt"
    "log"
    "io/ioutil"
)

func main() {
    config, err := ioutil.ReadFile("shipperFile")

    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf(string(config))

}
