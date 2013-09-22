SET GOPATH=%CD%
go install skeleton
go install gatekeeper
go install orchestrator
cp bin/orchestrator containers/orchestrator/orchestrator
SET VAGRANT_CWD=%CD%/test
vagrant up
cd test/skeleton/hello/ && go build
go test skeleton security orchestrator
