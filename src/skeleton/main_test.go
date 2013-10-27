package main

import (
        "io/ioutil"
        "net/http"
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
                t.Fatal(err)
        }

        cmd := exec.Command(os.Getenv("GOPATH")+"/bin/skeleton", "deploy")

        pipe, err := cmd.StdinPipe()
        if err != nil {
                t.Fatal(err)
        }

        cmd.Stdout = &testPrinter{t}
        cmd.Stderr = &testPrinter{t}

        pipe.Write([]byte("vagrant\n"))
        pipe.Write([]byte("vagrant\n"))

        err = cmd.Start()
        if err != nil {
                t.Fatal(err)
        }

        t.Log("wait")
        err = cmd.Wait()
        if err != nil {
                t.Fatal(err)
        }

        a := []string{"192.168.22.32", "192.168.22.33",
                "192.168.22.34", "192.168.22.35"}

        for _, address := range a {
                resp, err := http.Get("http://" + address)
                if err != nil {
                        t.Error(err.Error())
                        continue
                }
                b, err := ioutil.ReadAll(resp.Body)
                if err != nil {
                        t.Error(err.Error())
                        continue
                }
                if string(b) != "hello world\n" {
                        t.Error("Output is not hello world: \"" + string(b) + "\"")
                }
        }
}