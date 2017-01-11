package shelltest

import "regexp"

type (
	// A Result captures the output of a shell run
	Result struct {
		Script                    string
		Exit                      int
		Stdout, Stderr, Env, Errs string
	}
)

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
