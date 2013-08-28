all: bin/skeleton bin/security bin/orchestrator

ssh:
	GOPATH=$(CURDIR) go get -u code.google.com/p/go.crypto/ssh

bin/skeleton: src/skeleton/* ssh
	GOPATH=$(CURDIR) go install skeleton

bin/security: src/security/*
	GOPATH=$(CURDIR) go install security

bin/orchestrator: src/orchestrator/*
	GOPATH=$(CURDIR) go install orchestrator

vagrant:
	VAGRANT_CWD=$(CURDIR)/test vagrant up

test: all vagrant
	GOPATH=$(CURDIR) go test skeleton security orchestrator

.PHONY: all test ssh vagrant
