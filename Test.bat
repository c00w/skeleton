SET GOPATH=%CD%
go install skeleton
SET GOOS=linux
SET GOPATH=amd64
go install gatekeeper
go install orchestrator
copy bin\orchestrator containers\orchestrator\orchestrator
SET VAGRANT_CWD=%CD%\test
vagrant up
cd test\skeleton\hello\ && go build
SET GOOS=
SET GOPATH=
go test skeleton
go test security orchestrator
pause
