package hy

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/pkg/errors"
)

// FileWriter is something that can write Targets as files.
type FileWriter interface {
	// WriteFile writes a file representing target, by joining prefix with
	// Target.Path()
	WriteFile(prefix string, target WriteTarget) error
}

// FileReader reads data from prefix + target.Path() into target.Data.
type FileReader interface {
	ReadFile(prefix, filePath string, v interface{}) error
}

// FileMarshaler knows how to turn FileTargets into real files.
type FileMarshaler struct {
	// MarshalFunc is called to marshal values to bytes.
	MarshalFunc func(interface{}) ([]byte, error)
	// UnmarshalFunc is called to matshal bytes to values.
	UnmarshalFunc func([]byte, interface{}) error
	// FileExtension is the extension of files and should correspond to the byte
	// format written and read by MarshalFunc and UnmarshalFunc.
	FileExtension,
	// RootFileName is the name of the root struct, which will be written only
	// if the root is a struct with ordinary fields (not in a file or dir).
	RootFileName string
}

// JSONWriter is a FileWriter configured to marshal JSON.
var JSONWriter = FileMarshaler{
	MarshalFunc:   json.Marshal,
	UnmarshalFunc: json.Unmarshal,
	FileExtension: "json",
	RootFileName:  "_",
}

// ReadFile reads a file at prefix + t.Path into v.
func (fm FileMarshaler) ReadFile(prefix, filePath string, v interface{}) error {
	if filePath == "" {
		filePath = fm.RootFileName
	}
	filePath = filepath.Join(prefix, filePath)
	b, err := ioutil.ReadFile(filePath + "." + fm.FileExtension)
	if err != nil {
		return errors.Wrapf(err, "reading target file %q", filePath)
	}
	if err := fm.UnmarshalFunc(b, v); err != nil {
		return errors.Wrapf(err, "unmarshaling %q", filePath)
	}
	return nil
}

// WriteFile writes a file based on t.
func (fm FileMarshaler) WriteFile(prefix string, t WriteTarget) error {
	p := t.Path()
	if p == "" {
		p = fm.RootFileName
	}
	p = path.Join(prefix, p+"."+fm.FileExtension)
	dir := path.Dir(p)
	if dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.Wrapf(err, "creating directory %q", dir)
		}
	}
	b, err := fm.MarshalFunc(t.Data())
	if err != nil {
		return errors.Wrapf(err, "marshalling data")
	}
	return errors.Wrapf(ioutil.WriteFile(p, b, 0644), "writing file")
}
