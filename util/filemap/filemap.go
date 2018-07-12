package filemap

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// FileMap is a map of paths to file contents.
type FileMap map[string]string

// Mine returns true if the file at path relative to dir belongs to this
// FileMap. Otherwise it returns false.
func (f FileMap) Mine(dir, path string) bool {
	relPath, err := filepath.Rel(dir, path)
	if err != nil {
		return false
	}
	_, ok := f[relPath]
	return ok
}

// Merge attempts to merge 2 filemaps, if any keys conflict it panics.
func (f FileMap) Merge(o FileMap) FileMap {
	n := FileMap{}
	for k, v := range f {
		n[k] = v
	}
	for k, v := range o {
		if _, exists := n[k]; exists {
			panic(fmt.Sprintf("merging filemaps failed: duplicate key: %q", k))
		}
		n[k] = v
	}
	return n
}

// Clean deletes each file defined in f, it leaves any created directories
// in place. If you want to nuke everything, just use os.RemoveAll(dir).
func (f FileMap) Clean(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !f.Mine(dir, path) || info.IsDir() {
			return nil
		}
		return os.Remove(path)
	})
}

// Write writes the file tree defined in f to the directory dir.
// If dir exists and is not empty, or if any errors occur trying to write
// the file, or create the directory hierarchy, an error is returned, an
// f.Clean() is called to attempt to clean up any files written.
func (f FileMap) Write(dir string) error {
	for relPath, contents := range f {
		fullPath := filepath.Join(dir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0777); err != nil {
			return err
		}
		// Check the file doesn't exist.
		if _, err := os.Stat(fullPath); err == nil {
			return fmt.Errorf("file %q already exists", fullPath)
		} else if !os.IsNotExist(err) {
			return err
		}
		// Write the file.
		if err := ioutil.WriteFile(fullPath, []byte(contents), 0777); err != nil {
			if cleanErr := f.Clean(dir); err != nil {
				return errors.Wrapf(err, "error cleaning up: %s", cleanErr)
			}
			return errors.Wrapf(err, "error writing files, cleanup successful")
		}
	}
	return nil
}

// Session encapsulates tearing-up the file tree defined by f, then running some
// code that assumes it exists, before cleaning up. If f.Write(dir) fails, do is
// not run, and the error is returned. Otherwise, do is run, and the error from
// f.Clean(dir) is returned. If do panics, then f.Clean will not be run, so you
// can inspect the files created in case they had something to do with it.
func (f FileMap) Session(dir string, do func()) error {
	if err := f.Write(dir); err != nil {
		return err
	}
	do()
	return f.Clean(dir)
}
