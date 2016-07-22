package sous

import (
	"fmt"
	"strings"

	"github.com/samsalisbury/semv"
	"golang.org/x/text/unicode/norm"
)

type (
	// RepoURL is a URL to a source code repository.
	RepoURL string
	// RepoOffset is a path within a repository containing a single piece of
	// software.
	RepoOffset string
	// SourceID identifies a specific snapshot of a body of source code,
	// including its location and version.
	SourceID struct {
		RepoURL    RepoURL
		Version    semv.Version
		RepoOffset RepoOffset `yaml:",omitempty"`
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

func (sv SourceID) String() string {
	if sv.RepoOffset == "" {
		return fmt.Sprintf("%s %s", sv.RepoURL, sv.Version)
	}
	return fmt.Sprintf("%s:%s %s", sv.RepoURL, sv.RepoOffset, sv.Version)
}

// RevID returns the revision id for this SourceID.
func (sv SourceID) RevID() string {
	return sv.Version.Meta
}

// TagName returns the tag name for this SourceID.
func (sv SourceID) TagName() string {
	return sv.Version.Format("M.m.p-?")
}

// SourceLocation returns the location component of this SourceID
func (sv SourceID) SourceLocation() SourceLocation {
	return SourceLocation{
		RepoURL:    sv.RepoURL,
		RepoOffset: sv.RepoOffset,
	}
}

// Equal tests the equality between this SV and another
func (sv SourceID) Equal(o SourceID) bool {
	return sv.RepoURL == o.RepoURL && sv.RepoOffset == o.RepoOffset && sv.Version.Equals(o.Version)
}

// Repo returns the repository URL for this SV
func (sv SourceID) Repo() RepoURL {
	return sv.RepoURL
}

// DefaultDelim is a comma
const DefaultDelim = ","

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
	repoURL := RepoURL(chunks[0])
	version, err := semv.Parse(string(chunks[1]))
	if err != nil {
		return SourceID{}, err
	}
	repoOffset := RepoOffset("")
	if len(chunks) > 2 {
		repoOffset = RepoOffset(chunks[2])
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
	repoURL := RepoURL(chunks[0])
	repoOffset := RepoOffset("")
	if len(chunks) > 1 {
		repoOffset = RepoOffset(chunks[1])
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
