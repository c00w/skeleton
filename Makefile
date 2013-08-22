bin/shipper: src/shipper/*
	GOPATH=$(CURDIR) go install shipper

bin/security: src/security/*
	GOPATH=$(CURDIR) go install security

bin/orchestrator: src/orchestrator/*
	GOPATH=$(CURDIR) go install orchestrator

all: bin/shipper bin/security bin/orchestrator

dependencies: goyaml

goyaml:
	go get launchpad.net/goyaml

.PHONY: all goyaml dependencies
