all: bin/skeleton bin/security bin/orchestrator containers/orchestrator/orchestrator

containers/orchestrator/orchestrator: bin/orchestrator
	cp bin/orchestrator containers/orchestrator/orchestrator

bin/skeleton: src/skeleton/* src/common/*
	GOPATH=$(CURDIR) go install skeleton

bin/security: src/security/* src/common/*
	GOPATH=$(CURDIR) go install security

bin/orchestrator: src/orchestrator/* src/common/*
	GOPATH=$(CURDIR) go install orchestrator

vagrant:
	VAGRANT_CWD=$(CURDIR)/test vagrant up

test/skeleton/hello/hello: test/skeleton/hello/hello.go
	cd test/skeleton/hello/ && go build

test: all vagrant test/skeleton/hello/hello
	GOPATH=$(CURDIR) go test skeleton security orchestrator

clean:
	VAGRANT_CWD=$(CURDIR)/test vagrant destroy -f
.PHONY: all test vagrant
