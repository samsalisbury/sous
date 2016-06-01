// whitespace centralises information and some utility functions regarding
// whitespace.
package whitespace

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// Chars is a string containing all the literal whitespace characters.
	Chars = " \t\r\n"
	// EscapedChars is a string containing the whitespace characters escaped
	// with a backslash.
	EscapedChars = ` \t\r\n`
)

var (
	ws = regexp.MustCompile(fmt.Sprintf("[%s]+", EscapedChars))
)

// Regexp returns a compiled *regexp.Regexp representing whitespace.
func Regexp() *regexp.Regexp {
	return &(*ws)
}

// Trim trims all leading and trailing whitespace from a string.
func Trim(s string) string {
	return strings.Trim(s, Chars)
}

// Split splits a string into chunks by contiguous blocks of whitespace.
func Split(s string) []string {
	return ws.Split(s, -1)
}

// SplitN is similar to Split, but lets you specify the maximum number of chunks
// to return.
func SplitN(s string, n int) []string {
	return ws.Split(s, n)
}

// CleanWS takes a long string and left aligns it
// In other words, if you have a string in the midst of code, and all of its
// lines have at least the same level of indent as the first line, CleanWS will
// strip that indent from each line. There's an exception made for otherwise
// blank lines so that you don't need to maintain lines with only whitespace in
// the code.
func CleanWS(doc string) string {
	lines := strings.Split(doc, "\n")
	if len(lines) < 2 {
		return doc
	}
	for len(lines[0]) == 0 {
		lines = lines[1:]
	}

	for {
		tryLines := make([]string, 0, len(lines))
		first := lines[0]

		indent := first[0]

		for idx := range lines {
			if len(lines[idx]) == 0 {
				tryLines = append(tryLines, lines[idx])
				continue
			}
			if indent != lines[idx][0] {
				return strings.Join(lines, "\n")
			}
			tryLines = append(tryLines, lines[idx][1:])
		}
		lines = tryLines
	}
}
