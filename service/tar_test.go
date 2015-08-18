package service

import (
	"archive/tar"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"
	"testing"
)

func TestTar(t *testing.T) {
	buf := new(bytes.Buffer)
	err := Tar(".", buf)
	if err != nil {
		t.Fatal(err)
	}
	r := tar.NewReader(buf)
	for {
		_, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
	}
}

func TestUntar(t *testing.T) {
	buf := new(bytes.Buffer)
	err := Tar(".", buf)
	if err != nil {
		t.Fatal(err)
	}
	baseDir, err := ioutil.TempDir(os.TempDir(), "raftkv")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(baseDir)
	err = Untar(baseDir, tar.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}

}
