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

func (self *NamedVersion) RevId() string {
	return self.Version.Meta
}

func (self *NamedVersion) TagName() string {
	return self.Version.Format("M.m.s-?")
}

func (self *NamedVersion) CanonicalName() (cn CanonicalName) {
	cn.RepositoryName = self.RepositoryName
	cn.Path = self.Path
	return cn
}

func (self NamedVersion) Repo() RepositoryName {
	return self.RepositoryName
}

func (self CanonicalName) Repo() RepositoryName {
	return self.RepositoryName
}

func (self *CanonicalName) NamedVersion(version semv.Version) (nv NamedVersion) {
	nv.RepositoryName = self.RepositoryName
	nv.Path = self.Path
	nv.Version = version
	return nv
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
	if !(len(chunks[0]) > 0) {
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

	if !(len(chunks[0]) > 0) {
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

func ParseNamedVersion(source string) (nv NamedVersion, err error) {
	chunks := parseChunks(source)
	return namedVersionFromChunks(source, chunks)
}

func ParseCanonicalName(source string) (cn CanonicalName, err error) {
	chunks := parseChunks(source)
	return canonicalNameFromChunks(source, chunks)
}

func ParseGenName(source string) (name EntityName, err error) {
	chunks := parseChunks(source)
	switch len(chunks) {
	case 3:
		return namedVersionFromChunks(source, chunks)
	case 2:
		return canonicalNameFromChunks(source, chunks)
	default:
		return nil, fmt.Errorf("cannot parse %q - divides into %d chunks", source, len(chunks))
	}
	return
}
