package tgz

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/util"
)

const (
	useDefaultTempDir = ""
	tmpPrefix         = "tmp-tgz-"
)

// Extract decompress a gziped tarball into a new temporal directory
// created just for this purpose.
//
// On success, the path of the newly created directory and a nil error
// is returned.
//
// A non-nil error is returned if the method fails to complete. The
// returned path will be an empty string if no information was extracted
// before the error and the temporal directory has not been created.
// Otherwise, a non-empty string with the temporal directory holding
// whatever information was extracted before the error is returned.
func Extract(fs billy.Filesystem, tgz string) (d billy.Filesystem, err error, cleanup func()) {
	dirName := ""
	cleanup = func() {
		if dirName != "" {
			_ = os.RemoveAll(dirName)
		}
	}

	f, err := fs.Open(tgz)
	if err != nil {
		return
	}

	defer func() {
		errClose := f.Close()
		if err == nil {
			err = errClose
		}
	}()

	dirName, err = util.TempDir(fs, useDefaultTempDir, tmpPrefix)
	if err != nil {
		return
	}

	tar, err := zipTarReader(f)
	if err != nil {
		return
	}

	if err = unTar(fs, tar, dirName); err != nil {
		return
	}

	d, err = fs.Chroot(dirName)
	return
}

func zipTarReader(r io.Reader) (*tar.Reader, error) {
	zip, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return tar.NewReader(zip), nil
}

func unTar(fs billy.Filesystem, src *tar.Reader, dstPath string) error {
	for {
		header, err := src.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		dst := dstPath + "/" + header.Name
		mode := os.FileMode(header.Mode)
		switch header.Typeflag {
		case tar.TypeDir:
			err := fs.MkdirAll(dst, mode)
			if err != nil {
				return err
			}
		case tar.TypeReg:
			err := makeFile(fs, dst, mode, src)
			if err != nil {
				return err
			}
		default:
			return fmt.Errorf("Unable to untar type : %c in file %s",
				header.Typeflag, header.Name)
		}
	}

	return nil
}

func makeFile(fs billy.Filesystem, path string, mode os.FileMode, contents io.Reader) (err error) {
	w, err := fs.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		errClose := w.Close()
		if err == nil {
			err = errClose
		}
	}()

	_, err = io.Copy(w, contents)
	if err != nil {
		return err
	}

	if fs, ok := fs.(billy.Change); ok {
		if err = fs.Chmod(path, mode); err != nil {
			return err
		}
	}

	return nil
}
