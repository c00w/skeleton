package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestBonesLoading(t *testing.T) {
	err := os.Chdir(os.Getenv("GOPATH") + "/test/skeleton")
	if err != nil {
		t.Error(err)
	}
	cmd := exec.Command(os.Getenv("GOPATH") + "/bin/skeleton")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Log(string(out))
		t.Error(err)
	}
}
