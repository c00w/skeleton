SET GOPATH=%CD%
go install skeleton
SET GOOS=linux
SET GOPATH=amd64
go install gatekeeper
go install orchestrator
copy bin\orchestrator.exe containers\orchestrator\orchestrator
SET VAGRANT_CWD=%CD%\test
vagrant up
set GOPATH=test\skeleton\hello\
go build
SET GOOS=
set GOARCH=
SET GOPATH=%CD%
go test skeleton
go test security orchestrator
pause
