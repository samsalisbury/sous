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
