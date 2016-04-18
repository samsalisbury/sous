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
	// RepoOffsetDir is a path within a repository containing a single piece of
	// software.
	RepoOffset string
	// SourceLocation identifies a directory inside a source code repository.
	// Note that the directory has no meaning without the addition of a revision
	// ID. This type is used as a shorthand for deploy manifests, enabling the
	// logical grouping of deploys of different versions of a particular
	// service.
	SourceLocation struct {
		// RepoURL is the URL of a source code repository.
		RepoURL
		// RepoOffset is a relative path to a directory within the repository
		// at RepoURL
		RepoOffset
	}
	// SourceVersion is similar to SourceLocation except that it also includes
	// version information. This means that a SourceID completely describes
	// exactly one snapshot of a body of source code, from which a piece of
	// software can be built.
	SourceVersion struct {
		RepoURL
		semv.Version
		RepoOffset
	}

	EntityName interface {
		Repo() RepoURL
	}

	//Errors
	MissingRepo struct {
		parsing string
	}

	MissingVersion struct {
		repo    string
		parsing string
	}

	MissingPath struct {
		repo    string
		parsing string
	}

	IncludesVersion struct {
		parsing string
	}
)

func (nv *SourceVersion) RevId() string {
	return nv.Version.Meta
}

func (nv *SourceVersion) TagName() string {
	return nv.Version.Format("M.m.s-?")
}

func (nv *SourceVersion) CanonicalName() SourceLocation {
	return SourceLocation{
		RepoURL:    nv.RepoURL,
		RepoOffset: nv.RepoOffset,
	}
}

func (nv SourceVersion) Repo() RepoURL {
	return nv.RepoURL
}

func (cn SourceLocation) Repo() RepoURL {
	return cn.RepoURL
}

func (cn *SourceLocation) NamedVersion(version semv.Version) SourceVersion {
	return SourceVersion{
		RepoURL:    cn.RepoURL,
		RepoOffset: cn.RepoOffset,
		Version:    version,
	}
}

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

func namedVersionFromChunks(source string, chunks []string) (nv SourceVersion, err error) {
	if len(chunks[0]) == 0 {
		err = &MissingRepo{source}
		return
	}

	nv.RepoURL = RepoURL(chunks[0])

	nv.Version, err = semv.Parse(string(chunks[1]))
	if err != nil {
		return
	}
	if len(chunks) < 3 {
		nv.RepoOffset = ""
	} else {
		nv.RepoOffset = RepoOffset(chunks[2])
	}

	return
}

func canonicalNameFromChunks(source string, chunks []string) (cn SourceLocation, err error) {
	if len(chunks) > 2 {
		err = &IncludesVersion{source}
		return
	}

	if len(chunks[0]) == 0 {
		err = &MissingRepo{source}
		return
	}
	cn.RepoURL = RepoURL(chunks[0])

	if len(chunks) < 2 {
		cn.RepoOffset = ""
	} else {
		cn.RepoOffset = RepoOffset(chunks[1])
	}

	return
}

func ParseNamedVersion(source string) (SourceVersion, error) {
	chunks := parseChunks(source)
	return namedVersionFromChunks(source, chunks)
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
		return namedVersionFromChunks(source, chunks)
	case 2:
		return canonicalNameFromChunks(source, chunks)
	}
}
