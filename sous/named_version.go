package sous

import (
	"fmt"
	"strings"

	"github.com/samsalisbury/semv"
	"golang.org/x/text/unicode/norm"
)

type (
	RepositoryName string
	Path           string

	CanonicalName struct {
		RepositoryName
		Path
	}

	NamedVersion struct {
		RepositoryName
		semv.Version
		Path
	}

	EntityName interface {
		Repo() RepositoryName
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

func (nv *NamedVersion) RevId() string {
	return nv.Version.Meta
}

func (nv *NamedVersion) TagName() string {
	return nv.Version.Format("M.m.s-?")
}

func (nv *NamedVersion) CanonicalName() CanonicalName {
	return CanonicalName{
		RepositoryName: nv.RepositoryName,
		Path:           nv.Path,
	}
}

func (nv NamedVersion) Repo() RepositoryName {
	return nv.RepositoryName
}

func (cn CanonicalName) Repo() RepositoryName {
	return cn.RepositoryName
}

func (cn *CanonicalName) NamedVersion(version semv.Version) NamedVersion {
	return NamedVersion{
		RepositoryName: cn.RepositoryName,
		Path:           cn.Path,
		Version:        version,
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

func namedVersionFromChunks(source string, chunks []string) (nv NamedVersion, err error) {
	if len(chunks[0]) == 0 {
		err = &MissingRepo{source}
		return
	}

	nv.RepositoryName = RepositoryName(chunks[0])

	nv.Version, err = semv.Parse(string(chunks[1]))
	if err != nil {
		return
	}
	if len(chunks) < 3 {
		nv.Path = ""
	} else {
		nv.Path = Path(chunks[2])
	}

	return
}

func canonicalNameFromChunks(source string, chunks []string) (cn CanonicalName, err error) {
	if len(chunks) > 2 {
		err = &IncludesVersion{source}
		return
	}

	if len(chunks[0]) == 0 {
		err = &MissingRepo{source}
		return
	}
	cn.RepositoryName = RepositoryName(chunks[0])

	if len(chunks) < 2 {
		cn.Path = ""
	} else {
		cn.Path = Path(chunks[1])
	}

	return
}

func ParseNamedVersion(source string) (NamedVersion, error) {
	chunks := parseChunks(source)
	return namedVersionFromChunks(source, chunks)
}

func ParseCanonicalName(source string) (CanonicalName, error) {
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
