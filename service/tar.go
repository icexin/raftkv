package service

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"
)

func Tar(base string, w io.Writer) error {
	tw := tar.NewWriter(w)
	err := filepath.Walk(base, func(path string, info os.FileInfo, err error) error {
		name, _ := filepath.Rel(base, path)
		if name == "." {
			return nil
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = name

		tw.WriteHeader(hdr)
		if info.IsDir() {
			return nil
		}

		// write file content
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, f)
		f.Close()
		return err
	})
	if err != nil {
		return err
	}
	return tw.Close()

}

func Untar(base string, r io.Reader) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		info := hdr.FileInfo()
		fullpath := filepath.Join(base, info.Name())

		// as dir
		if info.IsDir() {
			os.MkdirAll(fullpath, 0755)
		}
		dir := filepath.Dir(fullpath)
		os.MkdirAll(dir, 0755)

		// as file
		f, err := os.Create(fullpath)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, tr)
		if err != nil {
			f.Close()
			return err
		}
		f.Chmod(info.Mode())
		f.Close()
	}
}
