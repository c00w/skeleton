all: bin/skeleton bin/security bin/orchestrator containers/orchestrator.tar.gz

containers/orchestrator/orchestrator: bin/orchestrator
	cp bin/orchestrator containers/orchestrator/orchestrator

containers/orchestrator.tar.gz: containers/orchestrator/orchestrator containers/orchestrator/*
	tar -cvf containers/orchestrator.tar.gz -C containers/orchestrator $(shell ls containers/orchestrator)

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

clean:
	VAGRANT_CWD=$(CURDIR)/test vagrant destroy -f
.PHONY: all test vagrant
