package common
	
import (
	"os"
	"testing"
	"io"
	"log"
	"os/exec"
)

func TestTarDir(t *testing.T){

	err := os.Chdir(os.Getenv("GOPATH") + "/test/tartest/")
	if err != nil {
		t.Error(err)
	}
	//run tarDir
	tar := TarDir("t0")

	//save it
	file, err := os.Create("test.tar.gz") 
	if err != nil {
		log.Fatal(err)
	}
	_, err = io.Copy(file,tar)

	err = file.Close()
	if err != nil {
		log.Fatal(err)
	}
	
	//extract it
	cmd := exec.Command("tar", "-xvf", "test.tar.gz")
    _, err = cmd.Output()

    if err != nil {
        println(err.Error())
        return
    }
    //check each of the files in the test directory
    _, err = os.Open("t1")
	if err != nil {
		log.Fatal(err)
	}
	_, err = os.Open("t1/t2")
	if err != nil {
		log.Fatal(err)
	}
	_, err = os.Open("t1/t2.txt")
	if err != nil {
		log.Fatal(err)
	}
	_, err = os.Open("t1/t2/t3.txt")
	if err != nil {
		log.Fatal(err)
	}

	//erase the extracted files / tarfile
	cmd = exec.Command("rm", "-rf", "t1")
    _, err = cmd.Output()

    if err != nil {
        println(err.Error())
        return
    }
    cmd = exec.Command("rm", "test.tar.gz")
    _, err = cmd.Output()

    if err != nil {
        println(err.Error())
        return
    }
}
