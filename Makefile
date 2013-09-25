all: bin/skeleton bin/gatekeeper bin/orchestrator containers/orchestrator/orchestrator containers/gatekeeper/gatekeeper

containers/orchestrator/orchestrator: bin/orchestrator
	cp bin/orchestrator containers/orchestrator/orchestrator

containers/gatekeeper/gatekeeper: bin/gatekeeper
	cp bin/gatekeeper containers/gatekeeper/gatekeeper

bin/skeleton: src/skeleton/*.go src/common/*.go
	GOPATH=$(CURDIR) go install skeleton

bin/gatekeeper: src/gatekeeper/*.go src/common/*.go
	GOPATH=$(CURDIR) go install gatekeeper

bin/orchestrator: src/orchestrator/*.go src/common/*.go
	GOPATH=$(CURDIR) go install orchestrator

vagrant:
	VAGRANT_CWD=$(CURDIR)/test vagrant up

test/skeleton/hello/hello: test/skeleton/hello/hello.go
	cd test/skeleton/hello/ && go build

test: all vagrant test/skeleton/hello/hello
	GOPATH=$(CURDIR) go test skeleton gatekeeper orchestrator

clean:
	VAGRANT_CWD=$(CURDIR)/test vagrant destroy -f

.PHONY: all test vagrant clean
