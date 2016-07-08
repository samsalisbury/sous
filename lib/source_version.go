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
	// SourceVersion is similar to SourceLocation except that it also includes
	// version information. This means that a SourceID completely describes
	// exactly one snapshot of a body of source code, from which a piece of
	// software can be built.
	SourceVersion struct {
		RepoURL    RepoURL
		Version    semv.Version
		RepoOffset RepoOffset `yaml:",omitempty"`
	}

	// EntityName is an interface over items with an arbitrary source repository
	EntityName interface {
		Repo() RepoURL
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

// MarshalYAML serializes this SourceLocation to a YAML document.
func (sl SourceLocation) MarshalYAML() (interface{}, error) {
	return sl.String(), nil
}

// UnmarshalYAML deserializes a YAML document into this SourceLocation
func (sl *SourceLocation) UnmarshalYAML(unmarshal func(interface{}) error) error {
	s := ""
	if err := unmarshal(&s); err != nil {
		return err
	}
	var err error
	*sl, err = ParseCanonicalName(s)
	return err
}

func (sl SourceLocation) String() string {
	if sl.RepoOffset == "" {
		return fmt.Sprintf("%s", sl.RepoURL)
	}
	return fmt.Sprintf("%s:%s", sl.RepoURL, sl.RepoOffset)
}

// Repo return the repository URL for this SourceLocation
func (sl SourceLocation) Repo() RepoURL {
	return sl.RepoURL
}

// SourceVersion returns a SourceVersion built from this location with the addition of a version
func (sl *SourceLocation) SourceVersion(version semv.Version) SourceVersion {
	return SourceVersion{
		RepoURL:    sl.RepoURL,
		RepoOffset: sl.RepoOffset,
		Version:    version,
	}
}

func (sv SourceVersion) String() string {
	if sv.RepoOffset == "" {
		return fmt.Sprintf("%s %s", sv.RepoURL, sv.Version)
	}
	return fmt.Sprintf("%s:%s %s", sv.RepoURL, sv.RepoOffset, sv.Version)
}

// RevID returns the revision id for this SourceVersion
func (sv *SourceVersion) RevID() string {
	return sv.Version.Meta
}

// TagName returns the tag name for this SourceVersion
func (sv *SourceVersion) TagName() string {
	return sv.Version.Format("M.m.p-?")
}

// CanonicalName returns a stable and consistent name for this SourceLocation
func (sv *SourceVersion) CanonicalName() SourceLocation {
	return SourceLocation{
		RepoURL:    sv.RepoURL,
		RepoOffset: sv.RepoOffset,
	}
}

// Equal tests the equality between this SV and another
func (sv *SourceVersion) Equal(o SourceVersion) bool {
	return sv.RepoURL == o.RepoURL && sv.RepoOffset == o.RepoOffset && sv.Version.Equals(o.Version)
}

// Repo returns the repository URL for this SV
func (sv SourceVersion) Repo() RepoURL {
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

func sourceVersionFromChunks(source string, chunks []string) (sv SourceVersion, err error) {
	if len(chunks[0]) == 0 {
		err = &MissingRepo{source}
		return
	}

	sv.RepoURL = RepoURL(chunks[0])

	sv.Version, err = semv.Parse(string(chunks[1]))
	if err != nil {
		return
	}
	if len(chunks) < 3 {
		sv.RepoOffset = ""
	} else {
		sv.RepoOffset = RepoOffset(chunks[2])
	}

	return
}

func canonicalNameFromChunks(source string, chunks []string) (sl SourceLocation, err error) {
	if len(chunks) > 2 {
		err = &IncludesVersion{source}
		return
	}

	if len(chunks[0]) == 0 {
		err = &MissingRepo{source}
		return
	}
	sl.RepoURL = RepoURL(chunks[0])

	if len(chunks) < 2 {
		sl.RepoOffset = ""
	} else {
		sl.RepoOffset = RepoOffset(chunks[1])
	}

	return
}

func ParseSourceVersion(source string) (SourceVersion, error) {
	chunks := parseChunks(source)
	return sourceVersionFromChunks(source, chunks)
}

func ParseCanonicalName(source string) (SourceLocation, error) {
	chunks := parseChunks(source)
	return canonicalNameFromChunks(source, chunks)
}

func ParseGenName(source string) (EntityName, error) {
	switch chunks := parseChunks(source); len(chunks) {
	default:
		return nil, fmt.Errorf("cannot parse %q - divides into %d chunks", source, len(chunks))
	case 3:
		return sourceVersionFromChunks(source, chunks)
	case 2:
		return canonicalNameFromChunks(source, chunks)
	}
}
