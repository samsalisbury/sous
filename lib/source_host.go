package sous

import (
	"fmt"
)

// SourceHost represents a source code repository host.
type SourceHost interface {
	// CanParseSourceLocation returns true if this SourceHost should attempt to
	// parse the string. If CanParseSourceLocation returns true, Sous will not
	// attempt to parse using any other SourceHost.
	CanParseSourceLocation(string) bool
	// ParseSourceLocation parses a SourceLocation from a string.
	ParseSourceLocation(string) (SourceLocation, error)
	// Owns returns true if this SourceHost owns the provided SourceLocation.
	Owns(SourceLocation) bool
	// GetSource returns the source code for this SourceID.
	GetSource(SourceID) (Source, error)
}

// GenericHost implements SourceHost, and is used as a fallback when none of the
// other SourceHosts are compatible with a SourceID.
type GenericHost struct{}

// CanParseSourceLocation always returns true.
func (h GenericHost) CanParseSourceLocation(string) bool { return true }

// ParseSourceLocation defers to the global ParseSourceLocation.
func (h GenericHost) ParseSourceLocation(s string) (SourceLocation, error) {
	return ParseSourceLocation(s)
}

// Owns always returns true.
func (h GenericHost) Owns(SourceLocation) bool { return true }

// GetSource always returns an error, since there is no generic way to get
// source code.
func (h GenericHost) GetSource(id SourceID) (Source, error) {
	return Source{}, fmt.Errorf("sous does not know how to get source code for %q", id)
}
