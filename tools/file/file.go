package file

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/dir"
)

func Write(data []byte, pathFormat string, a ...interface{}) {
	p := dir.Resolve(pathFormat, a...)
	dir.EnsureExists(dir.DirName(p))
	err := ioutil.WriteFile(p, data, 0777)
	if err != nil {
		cli.Fatalf("unable to write file %s; %s", p, err)
	}
}

func Create(path string) *os.File {
	f, err := os.Create(path)
	if err != nil {
		cli.Fatalf("Unable to write to file: %s", err)
	}
	return f
}

func Find(glob string) []string {
	files, err := filepath.Glob(glob)
	if err != nil {
		cli.Fatalf("Unable to search for files matching %s; %s", glob, err)
	}
	return files
}

func WriteString(data interface{}, pathFormat string, a ...interface{}) {
	s := fmt.Sprint(data)
	Write([]byte(s), pathFormat, a...)
}

func WriteJSON(data interface{}, pathFormat string, a ...interface{}) {
	b, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		cli.Fatalf("Unable to marshal %T object to JSON: %s", data, err)
	}
	Write(b, pathFormat, a...)
}

func Exists(filePath string) bool {
	filePath = dir.Resolve(filePath)
	i, err := os.Stat(filePath)
	if err == nil {
		return !i.IsDir()
	}
	if !os.IsNotExist(err) {
		cli.Fatalf("Unable to determine if file exists at '%s'; %s", filePath, err)
	}
	return false
}

func Move(path, newpath string) bool {
	if !Exists(path) {
		return false
	}
	if err := os.Rename(path, newpath); err != nil {
		cli.Fatalf("Unable to rename file: %s", err)
	}
	return true
}

func Link(path, newPath string) bool {
	if !Exists(path) {
		return false
	}
	if err := os.Link(path, newPath); err != nil {
		cli.Fatalf("Unable to link file: %s", err)
	}
	return true
}

func Remove(path string) bool {
	if !Exists(path) {
		return false
	}
	if err := os.Remove(path); err != nil {
		cli.Fatalf("Unable to remove file: %s", err)
	}
	return true
}

func ReadString(pathFormat string, a ...interface{}) (string, bool) {
	b, err, _ := Read(pathFormat, a...)
	return string(b), err
}

func ReadJSON(v interface{}, pathFormat string, a ...interface{}) bool {
	b, exists, path := Read(pathFormat, a...)
	if !exists {
		return false
	}
	if err := json.Unmarshal(b, &v); err != nil {
		cli.Fatalf("Unable to parse JSON in %s as %T: %s", path, v, err)
	}
	if v == nil {
		cli.Fatalf("Unmarshalled nil")
	}
	return true
}

func Read(pathFormat string, a ...interface{}) ([]byte, bool, string) {
	path := dir.Resolve(pathFormat, a...)
	contents, err := ioutil.ReadFile(path)
	if err == nil {
		return contents, true, path
	}
	if os.IsNotExist(err) {
		return nil, false, path
	}
	cli.Fatalf("Unable to read file %s: %s", path, err)
	return nil, false, path
}
