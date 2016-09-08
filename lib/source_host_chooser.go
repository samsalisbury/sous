package sous

import "fmt"

// SourceHostChooser wraps a slice of SourceHosts and delegates to the most
// appropriate one for various tasks.
type SourceHostChooser struct {
	// SourceHosts is an ordered list of SourceHosts. The order is significant,
	// earlier SourceHosts beat later ones.
	SourceHosts []SourceHost
}

// ParseSourceLocation tries to parse a SourceLocation using the first
// SourceHost that returns true for CanParseSourceLocation(s).
//
// It returns an error if the chosen SouceHost returns an error, or if none of
// the SourceHosts return true for CanParseSourceLocation(s).
func (e *SourceHostChooser) ParseSourceLocation(s string) (SourceLocation, error) {
	for _, h := range e.SourceHosts {
		if h.CanParseSourceLocation(s) {
			return h.ParseSourceLocation(s)
		}
	}
	return SourceLocation{}, fmt.Errorf("source location not recognised: %q", s)
}
