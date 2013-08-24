bin/skeleton: src/skeleton/*
	GOPATH=$(CURDIR) go install skeleton

bin/security: src/security/*
	GOPATH=$(CURDIR) go install security

bin/orchestrator: src/orchestrator/*
	GOPATH=$(CURDIR) go install orchestrator

all: bin/skeleton bin/security bin/orchestrator

dependencies: goyaml

goyaml:
	go get launchpad.net/goyaml

.PHONY: all goyaml dependencies
