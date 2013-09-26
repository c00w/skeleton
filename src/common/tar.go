package common

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"io/ioutil"
	"log"
	"os"
	//"strings"
)

func tardir(path string, write_path string, totar map[*tar.Header][]byte, top bool) {
	// Find subdirectories
	ifd, err := os.Open(path)
	if err != nil {
		log.Fatal(err)
	}

	fi, err := ifd.Readdir(-1)
	if err != nil {
		log.Fatal(err)
	}

	// Put them in tarfile
	for _, f := range fi {
		if f.IsDir() {
			tardir(path+"/"+f.Name(), write_path+"/"+f.Name(), totar, false)
		} else {
			h, err := tar.FileInfoHeader(f, "")
			if err != nil {
				log.Fatal(err)
			}
			if !top {
				h.Name = write_path + "/" + h.Name
			}
			

			ffd, err := os.Open(path + "/" + f.Name())

			if err != nil {
				log.Fatal(err)
			}

			c, err := ioutil.ReadAll(ffd)
			if err != nil {
				log.Fatal(err)
			}
			totar[h] = c
		}
	}
}

// tarDir takes a directory path and produces a reader which is all of its
// contents tarred up and compressed with gzip
func TarDir(path string) io.Reader {
	// check this is a directory
	i, err := os.Stat(path)
	if err != nil {
		log.Fatal(err)
	}
	if !i.IsDir() {
		log.Fatal("Directory to tar up is not a directory")
	}

	//Make a buffer to hold the file
	b := bytes.NewBuffer(nil)
	g := gzip.NewWriter(b)
	w := tar.NewWriter(g)

	//fi is a 'slice' of all of the files/subdirectories at path
	totar := map[*tar.Header][]byte{}
	tardir(path, "", totar, true)

	//fmt.Println(totar)
	for k, v := range totar {
		w.WriteHeader(k)
		w.Write(v)
	}

	
	w.Close()
	g.Close()
	return b
}
