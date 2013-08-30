all: bin/skeleton bin/security bin/orchestrator

bin/skeleton: src/skeleton/*
	GOPATH=$(CURDIR) go install skeleton

bin/security: src/security/*
	GOPATH=$(CURDIR) go install security

bin/orchestrator: src/orchestrator/*
	GOPATH=$(CURDIR) go install orchestrator

vagrant:
	VAGRANT_CWD=$(CURDIR)/test vagrant up

test: all vagrant
	GOPATH=$(CURDIR) go test skeleton security orchestrator

.PHONY: all test vagrant
