SET GOPATH=%CD%
go install skeleton
go install gatekeeper
go install orchestrator
copy bin\orchestrator.exe containers\orchestrator\orchestrator
SET VAGRANT_CWD=%CD%\test
vagrant up
cd test\skeleton\hello\ && go build
go test skeleton security orchestrator
pause
