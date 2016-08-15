package sous

import (
	"fmt"

	"github.com/samsalisbury/semv"
)

type (
	// SourceLocation identifies a directory inside a specific source code repo.
	// Note that the directory has no meaning without the addition of a revision
	// ID. This type is used as a shorthand for deploy manifests, enabling the
	// logical grouping of deploys of different versions of a particular
	// service.
	SourceLocation struct {
		// RepoURL is the URL of a source code repository.
		RepoURL string
		// RepoOffset is a relative path to a directory within the repository
		// at RepoURL
		RepoOffset string `yaml:",omitempty"`
	}
)

// NewSourceLocation creates a new SourceLocation from strings.
func NewSourceLocation(repoURL, repoOffset string) SourceLocation {
	return SourceLocation{repoURL, repoOffset}
}

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
	*sl, err = ParseSourceLocation(s)
	return err
}

func (sl SourceLocation) String() string {
	if sl.RepoOffset == "" {
		return fmt.Sprintf("%s", sl.RepoURL)
	}
	return fmt.Sprintf("%s:%s", sl.RepoURL, sl.RepoOffset)
}

// Repo return the repository URL for this SourceLocation
func (sl SourceLocation) Repo() string {
	return sl.RepoURL
}

// SourceID returns a SourceID built from this location with the addition of a version.
func (sl *SourceLocation) SourceID(version semv.Version) SourceID {
	return SourceID{
		RepoURL:    sl.RepoURL,
		RepoOffset: sl.RepoOffset,
		Version:    version,
	}
}
