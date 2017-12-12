package shelltest

import (
	"os"
	"path/filepath"
	"regexp"
)

type (
	// A Result captures the output of a shell run
	Result struct {
		Script                             string
		Exit                               int
		Stdout, Stderr, Blended, Env, Errs string
	}
)

// WriteTo writes details of this result to a particular path
func (res Result) WriteTo(dir, base string) error {
	var allErr error
	if err := writePart(dir, base, "sh", res.Script); err != nil {
		allErr = err
	}

	if err := writePart(dir, base, "stdout", res.Stdout); err != nil {
		allErr = err
	}

	if err := writePart(dir, base, "stderr", res.Stderr); err != nil {
		allErr = err
	}

	if err := writePart(dir, base, "blended", res.Blended); err != nil {
		allErr = err
	}

	if err := writePart(dir, base, "errs", res.Errs); err != nil {
		allErr = err
	}

	if err := writePart(dir, base, "env", res.Env); err != nil {
		allErr = err
	}

	return allErr
}

func writePart(dir, base, ext, content string) (err error) {
	file, err := os.Create(filepath.Join(dir, base+"."+ext))
	if err != nil {
		return err
	}

	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	_, err = file.WriteString(content)
	return err
}

// StdoutMatches asserts that the stdout of the result matches a regex pattern
func (res Result) StdoutMatches(pattern string) bool {
	re := regexp.MustCompile(pattern)
	return re.MatchString(res.Stdout)
}

// StderrMatches asserts that the stdout of the result matches a regex pattern
func (res Result) StderrMatches(pattern string) bool {
	re := regexp.MustCompile(pattern)
	return re.MatchString(res.Stderr)
}

// Matches asserts that the stdout of the result matches a regex pattern
func (res Result) Matches(pattern string) bool {
	re := regexp.MustCompile(pattern)
	return re.MatchString(res.Stderr) || re.MatchString(res.Stdout)
}

// ErrsMatches asserts that the Errs stream matches a regex pattern - for instance to check that a particular command fails.
func (res Result) ErrsMatches(pattern string) bool {
	re := regexp.MustCompile(pattern)
	return re.MatchString(res.Errs)
}
