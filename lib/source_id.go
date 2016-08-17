package sous

import (
	"fmt"
	"strings"

	"github.com/samsalisbury/semv"
	"golang.org/x/text/unicode/norm"
)

type (
	// SourceID identifies a specific snapshot of a body of source code,
	// including its location and version.
	SourceID struct {
		RepoURL    string
		Version    semv.Version
		RepoOffset string `yaml:",omitempty"`
	}

	//MissingRepo indicates that Sous couldn't determine which repo was intended for this SL
	MissingRepo struct {
		parsing string
	}

	//MissingVersion indicates that Sous couldn't determine what version was intended for this SL
	MissingVersion struct {
		repo    string
		parsing string
	}

	//MissingPath indicates that Sous couldn't determine what repo offset was intended for this SL
	MissingPath struct {
		repo    string
		parsing string
	}

	//IncludesVersion indicates that Sous couldn't determine what version was intended for this SL
	IncludesVersion struct {
		parsing string
	}
)

// DefaultDelim is the default delimiter between parts of the string
// representation of a SourceID or a SourceLocation.
const DefaultDelim = ","

func (sid SourceID) String() string {
	if sid.RepoOffset == "" {
		return fmt.Sprintf("%s %s", sid.RepoURL, sid.Version)
	}
	return fmt.Sprintf("%s:%s %s", sid.RepoURL, sid.RepoOffset, sid.Version)
}

// Tag returns the version tag for this source ID.
func (sid SourceID) Tag() string {
	return sid.Version.Format(semv.MajorMinorPatch)
}

// RevID returns the revision id for this SourceID.
func (sid SourceID) RevID() string {
	return sid.Version.Meta
}

// Location returns the location component of this SourceID.
func (sid SourceID) Location() SourceLocation {
	return SourceLocation{
		RepoURL:    sid.RepoURL,
		RepoOffset: sid.RepoOffset,
	}
}

// Equal tests the equality between this SourceID and another.
func (sid SourceID) Equal(o SourceID) bool {
	return sid == o
}

func (err *IncludesVersion) Error() string {
	return fmt.Sprintf("Three parts found (includes a version?) in a canonical name: %q", err.parsing)
}

func (err *MissingRepo) Error() string {
	return fmt.Sprintf("No repository found in %q", err.parsing)
}

func (err *MissingVersion) Error() string {
	return fmt.Sprintf("No version found in %q (did find repo: %q)", err.parsing, err.repo)
}

func (err *MissingPath) Error() string {
	return fmt.Sprintf("No path found in %q (did find repo: %q)", err.parsing, err.repo)
}

func parseChunks(sourceStr string) []string {
	source := norm.NFC.String(sourceStr)

	delim := DefaultDelim
	if !('A' <= source[0] && source[0] <= 'Z') && !('a' <= source[0] && source[0] <= 'z') {
		delim = source[0:1]
		source = source[1:]
	}

	return strings.Split(source, delim)
}

func sourceIDFromChunks(source string, chunks []string) (SourceID, error) {
	if len(chunks[0]) == 0 {
		return SourceID{}, &MissingRepo{source}
	}
	repoURL := chunks[0]
	version, err := semv.Parse(string(chunks[1]))
	if err != nil {
		return SourceID{}, err
	}
	repoOffset := ""
	if len(chunks) > 2 {
		repoOffset = chunks[2]
	}
	return SourceID{
		Version:    version,
		RepoURL:    repoURL,
		RepoOffset: repoOffset,
	}, nil
}

func sourceLocationFromChunks(source string, chunks []string) (SourceLocation, error) {
	if len(chunks) > 2 {
		return SourceLocation{}, &IncludesVersion{source}
	}
	if len(chunks[0]) == 0 {
		return SourceLocation{}, &MissingRepo{source}
	}
	repoURL := chunks[0]
	repoOffset := ""
	if len(chunks) > 1 {
		repoOffset = chunks[1]
	}
	return SourceLocation{RepoURL: repoURL, RepoOffset: repoOffset}, nil
}

// ParseSourceID parses an entire SourceID.
func ParseSourceID(s string) (SourceID, error) {
	chunks := parseChunks(s)
	return sourceIDFromChunks(s, chunks)
}

// ParseSourceLocation parses an entire SourceLocation.
func ParseSourceLocation(s string) (SourceLocation, error) {
	chunks := parseChunks(s)
	return sourceLocationFromChunks(s, chunks)
}
