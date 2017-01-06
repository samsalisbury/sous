package sous

import (
	"fmt"

	"github.com/samsalisbury/semv"
)

type (
	// SourceLocation identifies a directory inside a source code repository.
	// Note that the directory is ambiguous without the addition of a revision
	// ID.
	// N.b. the {M,Unm}arshal* methods - SL doesn't serialize as you might expect
	SourceLocation struct {
		// Repo identifies a source code repository.
		Repo,
		// Dir is a directory within the repository at Repo containing the
		// source code for one piece of software.
		Dir string
	}
)

// NewSourceLocation creates a new SourceLocation from strings.
func NewSourceLocation(repoURL, repoOffset string) SourceLocation {
	return SourceLocation{repoURL, repoOffset}
}

// ParseSourceLocation parses an entire SourceLocation.
func ParseSourceLocation(s string) (SourceLocation, error) {
	chunks := parseChunks(s)
	return sourceLocationFromChunks(s, chunks)
}

// MustParseSourceLocation wraps ParseSourceLocation but panics instead of
// returning a non-nil error.
func MustParseSourceLocation(s string) SourceLocation {
	sl, err := ParseSourceLocation(s)
	if err != nil {
		panic(err)
	}
	return sl
}

func sourceLocationFromChunks(source string, chunks []string) (SourceLocation, error) {
	if len(chunks) > 2 {
		return SourceLocation{}, &IncludesVersion{source}
	}
	if len(chunks) == 0 || len(chunks[0]) == 0 {
		return SourceLocation{}, &MissingRepo{source}
	}
	repoURL := chunks[0]
	repoOffset := ""
	if len(chunks) > 1 {
		repoOffset = chunks[1]
	}
	return SourceLocation{Repo: repoURL, Dir: repoOffset}, nil
}

// MarshalYAML serializes this SourceLocation to a YAML document.
func (sl SourceLocation) MarshalYAML() (interface{}, error) {
	return sl.String(), nil
}

// MarshalText implements encoding.TextMarshaler.
func (sl SourceLocation) MarshalText() ([]byte, error) {
	return []byte(sl.String()), nil
}

// UnmarshalText implements encoding.TextMarshaler.
func (sl *SourceLocation) UnmarshalText(b []byte) error {
	var err error
	*sl, err = ParseSourceLocation(string(b))
	return err
}

// UnmarshalYAML deserializes a YAML document into this SourceLocation
func (sl *SourceLocation) UnmarshalYAML(unmarshal func(interface{}) error) error {
	s := ""
	if err := unmarshal(&s); err != nil {
		return err
	}
	var err error
	*sl, err = ParseSourceLocation(s)
	return err
}

func (sl SourceLocation) String() string {
	if sl.Dir == "" {
		return fmt.Sprintf("%s", sl.Repo)
	}
	return fmt.Sprintf("%s,%s", sl.Repo, sl.Dir)
}

// SourceID returns a SourceID built from this location with the addition of a version.
func (sl SourceLocation) SourceID(version semv.Version) SourceID {
	return SourceID{
		Location: sl,
		Version:  version,
	}
}
