package github

import (
	"fmt"
	"strings"

	"github.com/opentable/sous/lib"
)

// SourceHost is the GitHub source code host.
// It satisfies sous.SourceHost.
type SourceHost struct{}

// CanParseSourceLocation returns true if s begins with Prefix.
func (SourceHost) CanParseSourceLocation(s string) bool {
	return strings.HasPrefix(s, Prefix)
}

// ParseSourceLocation parses a GitHub source location.
func (SourceHost) ParseSourceLocation(s string) (sous.SourceLocation, error) {
	return ParseSourceLocation(s)
}

// Owns returns true if sl.Repo begins with Prefix.
func (SourceHost) Owns(sl sous.SourceLocation) bool {
	return strings.HasPrefix(sl.Repo, Prefix)
}

// GetSource returns a sous.Source for the provided id.
// It returns an error if Owns(id.Location) returns false, or
// if any internal operations, which may include network requests,
// disk access, etc, fail.
func (h SourceHost) GetSource(id sous.SourceID) (sous.Source, error) {
	if !h.Owns(id.Location) {
		return sous.Source{}, fmt.Errorf("the github source host cannot get source for %q",
			id.Location)
	}
	return sous.Source{}, fmt.Errorf("fetching from GitHub not yet supported")
}
