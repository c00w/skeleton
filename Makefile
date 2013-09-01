all: bin/skeleton bin/security bin/orchestrator containers/orchestrator.tar.gz

containers/orchestrator/orchestrator: bin/orchestrator
	cp bin/orchestrator containers/orchestrator/orchestrator

containers/orchestrator.tar.gz: containers/orchestrator/orchestrator
	tar -cvf containers/orchestrator.tar.gz containers/orchestrator

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
