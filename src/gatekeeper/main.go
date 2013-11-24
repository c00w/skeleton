package main

import (
	"libgatekeeper"
	"log"
)

func main() {

	g := libgatekeeper.NewServer()
	err := g.Listen(":800")
	log.Fatal(err)

}
