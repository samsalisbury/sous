package hy

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// FileTreeReader gets targets from the filesystem.
type FileTreeReader struct {
	// FileExtension is the extension of files to consider.
	FileExtension string
	// Prefix is the path prefix.
	Prefix string
	// RootFileName is the root file name.
	RootFileName string
}

// NewFileTreeReader returns a new FileTreeReader configured to consider files
// with extension ext.
func NewFileTreeReader(ext, rootFileName string) *FileTreeReader {
	return &FileTreeReader{
		FileExtension: ext,
		RootFileName:  rootFileName,
	}
}

// ReadTree reads a tree rooted at prefix and generates a target from each file
// with extension FileExtension found in the tree.
func (ftr *FileTreeReader) ReadTree(prefix string) (FileTargets, error) {
	ftr.Prefix = prefix
	targets := MakeFileTargets(0)
	if err := filepath.Walk(prefix, ftr.MakeWalkFunc(targets)); err != nil {
		return targets, errors.Wrapf(err, "walking tree")
	}
	return targets, nil
}

// MakeWalkFunc makes a func to process a single filesystem object.
func (ftr *FileTreeReader) MakeWalkFunc(targets FileTargets) filepath.WalkFunc {
	return func(p string, fi os.FileInfo, err error) error {
		if err != nil || fi.IsDir() || filepath.Ext(p) != "."+ftr.FileExtension {
			return err
		}
		path := strings.TrimPrefix(p, ftr.Prefix+"/")
		path = strings.TrimSuffix(path, "."+ftr.FileExtension)
		if path == ftr.RootFileName {
			path = ""
		}
		t := &FileTarget{
			FilePath: path,
		}
		return errors.Wrapf(targets.Add(t), "adding file target %q", p)
	}
}
