package main
import (
    "testing"
    "os"
    "os/exec"
)

func TestBonesLoading(t *testing.T) {
    err := os.Chdir(os.Getenv("GOPATH") + "/test/skeleton")
    if err != nil {
        t.Error(err)
    }
    cmd := exec.Command(os.Getenv("GOPATH") + "/bin/skeleton")
    err = cmd.Run()
    if err != nil {
        t.Error(err)
    }
}
