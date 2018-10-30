package sous

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
	"golang.org/x/text/unicode/norm"
)

type (
	// SourceID identifies a specific snapshot of a body of source code,
	// including its location and version.
	SourceID struct {
		// Location is the repo/dir pair indicating the location of the source
		// code. Note that not all locations will be valid with all Versions.
		Location SourceLocation
		// Version identifies a specific version of the source code at Repo/Dir.
		Version semv.Version
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

// MakeSourceID is a convenience function to build a SourceID.
func MakeSourceID(repo, dir, version string) SourceID {
	return SourceID{
		Location: SourceLocation{
			Repo: repo,
			Dir:  dir,
		},
		Version: semv.MustParse(version),
	}
}

// DefaultDelim is the default delimiter between parts of the string
// representation of a SourceID or a SourceLocation.
const DefaultDelim = ","

func (sid SourceID) String() string {
	if sid.Location.Dir == "" {
		return fmt.Sprintf("%s,%s", sid.Location.Repo, sid.Version)
	}
	return fmt.Sprintf("%s,%s,%s", sid.Location.Repo, sid.Version, sid.Location.Dir)
}

// QueryValues returns the url.Values for this SourceIDs
func (sid SourceID) QueryValues() url.Values {
	v := url.Values{}
	v.Set("repo", sid.Location.Repo)
	v.Set("offset", sid.Location.Dir)
	v.Set("version", sid.Version.String())
	return v
}

// EachField implements logging.EachFielder on SourceID.
func (sid SourceID) EachField(fn logging.FieldReportFn) {
	fn(logging.SousSourceId, sid.String())
	// XXX consider a version field - would need to be added to OTLs
	sid.Location.EachField(fn)
}

// Tag returns the version tag for this source ID.
func (sid SourceID) Tag() string {
	return sid.Version.Format(semv.MajorMinorPatch)
}

/*
// RevID returns the revision id for this SourceID.
func (sid SourceID) RevID() string {
	return sid.Version.Meta
}
*/

// Equal tests the equality between this SourceID and another.
func (sid SourceID) Equal(o SourceID) bool {
	if !sid.Version.Equals(o.Version) {
		return false
	}
	// Equalise the versions so we can do a simple equality test.
	// This is safe because sid and o are values not pointers.
	sid.Version = o.Version
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
	if len(sourceStr) == 0 {
		return []string{}
	}
	source := norm.NFC.String(sourceStr)

	delim := DefaultDelim
	if !('A' <= source[0] && source[0] <= 'Z') &&
		!('a' <= source[0] && source[0] <= 'z') &&
		!('0' <= source[0] && source[0] <= '9') &&
		!(source[0] == '.' || source[0] == '/') {
		delim = source[0:1]
		source = source[1:]
	}

	return strings.Split(source, delim)
}

func sourceIDFromChunks(source string, chunks []string) (SourceID, error) {
	if len(chunks[0]) == 0 {
		return SourceID{}, errors.Wrap(&MissingRepo{source}, "parsing")
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
		Location: SourceLocation{
			Dir:  repoOffset,
			Repo: repoURL,
		},
		Version: version,
	}, nil
}

// ParseSourceID parses an entire SourceID.
func ParseSourceID(s string) (SourceID, error) {
	chunks := parseChunks(s)
	return sourceIDFromChunks(s, chunks)
}

// MustParseSourceID wraps ParseSourceID and panics if it returns an error.
func MustParseSourceID(s string) SourceID {
	sid, err := ParseSourceID(s)
	if err != nil {
		panic(err)
	}
	return sid
}

// NewSourceID attempts to create a new SourceID from strings representing the
// separate components. It expects a repo to be in canonicalised form, e.g.
// host/some/path. It does not attempt to translate or validate the repo.
func NewSourceID(repo, offset, version string) (SourceID, error) {
	v, err := semv.Parse(version)
	if err != nil {
		return SourceID{}, err
	}
	return SourceID{
		Location: SourceLocation{
			Repo: repo, Dir: offset,
		},
		Version: v,
	}, nil
}

// MustNewSourceID wraps NewSourceID and panics if it returns an error.
func MustNewSourceID(repo, offset, version string) SourceID {
	sid, err := NewSourceID(repo, offset, version)
	if err != nil {
		panic(err)
	}
	return sid
}
