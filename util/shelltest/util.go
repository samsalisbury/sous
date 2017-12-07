package shelltest

import (
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// BuildPath constructs a PATH that includes the directories with given exectuables in them.
func BuildPath(exes ...string) (string, error) {
	dirMap := map[string]struct{}{}
	dirList := []string{}

	for _, name := range exes {
		exePath, err := exec.LookPath(name)
		if err != nil {
			return "", err
		}

		dir := filepath.Dir(exePath)

		if _, already := dirMap[dir]; !already {
			dirList = append(dirList, dir)
		}
		dirMap[dir] = struct{}{}
	}

	return strings.Join(dirList, ":"), nil
}

// TemplateConfigs walks the sourceDir, templating into targetDir based on configData
func TemplateConfigs(sourceDir, targetDir string, configData interface{}) error {
	log.Printf("Templating %q -> %q.", sourceDir, targetDir)
	var linkCount, templCount int
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return errors.Wrap(err, "open")
		}

		if 0 != (info.Mode() & os.ModeSymlink) {
			linkT, errLink := os.Readlink(path)

			if errLink != nil {
				return errors.Wrap(errLink, "readlink")
			}
			if filepath.IsAbs(linkT) {
				linkT, err = filepath.Rel(sourceDir, linkT)
				if err != nil {
					return errors.Wrap(err, "Rel link")
				}
			}
			linkName := filepath.Join(targetDir, info.Name())
			linkCount++
			return errors.Wrap(os.Symlink(linkT, linkName), "create link")
		}

		sourcePath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return errors.Wrap(err, "Rel file")
		}

		bytes, err := ioutil.ReadAll(f)
		if err != nil {
			return errors.Wrap(err, "read")
		}

		targetPath := filepath.Join(targetDir, sourcePath)
		err = os.MkdirAll(filepath.Dir(targetPath), os.ModePerm)
		if err != nil {
			return errors.Wrap(err, "create dir")
		}

		target, err := os.Create(targetPath)
		if err != nil {
			return errors.Wrap(err, "create target")
		}

		defer func() {
			if cerr := target.Close(); cerr != nil && err == nil {
				err = cerr
			}
		}()

		tmpl, err := template.New(f.Name()).Parse(string(bytes))
		if err != nil {
			return errors.Wrap(err, "parse")
		}

		templCount++
		return errors.Wrap(tmpl.Execute(target, configData), "execute")
	})
	log.Printf("Linked %d files, Templated %d files.", linkCount, templCount)
	return err
}

// WithHostEnv copies values from the host system into an env map, merging given values in.
func WithHostEnv(hostEnvs []string, env map[string]string) map[string]string {
	newEnv := make(map[string]string)
	for _, k := range hostEnvs {
		newEnv[k] = os.Getenv(k)
	}
	for k, v := range env {
		newEnv[k] = v
	}
	return newEnv
}
