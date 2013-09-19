package main
	
import (
	"os"
	"testing"
	"io"
	"common"
	"log"
)

func TestTarDir(t *testing.T){

	err := os.Chdir(os.Getenv("GOPATH") + "/test/tartest")
	if err != nil {
		t.Error(err)
	}

	tar := common.TarDir("t0")

	file, err := os.Create("test.tar.gz") 
	if err != nil {
		log.Fatal(err)
	}

	_, err = io.Copy(file,tar)

	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}
}
