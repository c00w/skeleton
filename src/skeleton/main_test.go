package main

import (
	"os"
	"os/exec"
	"testing"
)

type testPrinter struct {
	t *testing.T
}

func (t *testPrinter) Write(p []byte) (n int, err error) {
	x := string(p)
	if len(x) > 0 && x[len(x)-1] == '\n' {
		x = x[0 : len(x)-1]
	}
	t.t.Log("binary:" + x)
	return len(p), nil

}

func TestBonesLoading(t *testing.T) {
	err := os.Chdir(os.Getenv("GOPATH") + "/test/skeleton")
	if err != nil {
		t.Error(err)
	}

	cmd := exec.Command(os.Getenv("GOPATH") + "/bin/skeleton", "deploy" )

	pipe, err := cmd.StdinPipe()
	if err != nil {
		t.Error(err)
	}

	cmd.Stdout = &testPrinter{t}
	cmd.Stderr = &testPrinter{t}

	pipe.Write([]byte("vagrant\n"))
	pipe.Write([]byte("vagrant\n"))

	err = cmd.Start()
	if err != nil {
		t.Error(err)
	}

	t.Log("wait")
	err = cmd.Wait()
	if err != nil {
		t.Error(err)
	}
}
