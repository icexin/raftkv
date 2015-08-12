package service

import (
	"archive/tar"
	"bytes"
	"io"
	"log"
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
		hdr, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		t.Logf("%s", hdr.Name)
	}
}
