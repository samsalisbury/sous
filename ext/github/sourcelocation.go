package github

import (
	"fmt"
	"path"
	"strings"

	"github.com/opentable/sous/lib"
)

const (
	// Prefix is the prefix that all GitHub repository paths have.
	Prefix = "github.com/"
	// Alpha contains all the lower and upper case ASCII letters.
	Alpha = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// Numeric contains all the ASCII digits.
	Numeric = "0123456789"
	// Hyphen is an ASCII hyphen-minus sign -.
	Hyphen = "-"
	// Dot is an ASCII full stop.
	Dot = "."
	// Underscore is an ASCII underscore.
	Underscore = "_"
	// Slash is an ASCII slash or divide character (unicode solidus)
	Slash = "/"

	// UsernameAllowedChars are the characters allowed anywhere in a username.
	UsernameAllowedChars = Alpha + Numeric
	// UsernameAllowedMiddleChars are the characters allowed in the middle of a
	// username.
	UsernameAllowedMiddleChars = UsernameAllowedChars + Hyphen

	// RepoNameAllowedChars are the characters allowed anywhere in a repo name.
	RepoNameAllowedChars = Alpha + Numeric
	// RepoNameAllowedMiddleChars are the characters allowed in the middle of a
	// repo name.
	RepoNameAllowedMiddleChars = RepoNameAllowedChars + Hyphen + Dot + Underscore

	// OffsetAllowedChars are characters allowed in the offset directory path.
	OffsetAllowedChars = Alpha + Numeric + Hyphen + Dot + Underscore + Slash
)

// ParseSourceLocation parses s for a GitHub based sous.SourceLocation.
// It returns an error if the string is malformed, or if it does not begin with
// "github.com/".
func ParseSourceLocation(s string) (sous.SourceLocation, error) {
	if !strings.HasPrefix(s, Prefix) {
		return sous.SourceLocation{}, fmt.Errorf("%q does not begin with %q", s, Prefix)
	}
	repodir := strings.SplitN(s, ",", 2)
	rp, err := parseRepoPath(repodir[0])
	if err != nil {
		return sous.SourceLocation{}, err
	}
	if len(repodir) == 2 {
		if rp.Dir == "" {
			rp.Dir = repodir[1]
		} else {
			rp.Dir = rp.Dir + "," + repodir[1]
		}
	}
	if err := rp.Validate(); err != nil {
		return sous.SourceLocation{}, err
	}
	return sous.SourceLocation{
		Repo: path.Join(Prefix, rp.User, rp.Repo),
		Dir:  rp.Dir,
	}, nil
}

// RepoPath represents the known parts of a GitHub repository path.
type repoPath struct {
	// User is the GitHub username.
	User,
	// Repo is the name of the repository.
	Repo,
	// Dir is the offset directory of the source code in the repository.
	Dir string
}

func parseRepoPath(s string) (repoPath, error) {
	parts := strings.Split(s, "/")
	if len(parts) < 3 {
		return repoPath{}, fmt.Errorf("%q does not identify a repository", s)
	}
	rp := repoPath{
		User: parts[1],
		Repo: parts[2],
		Dir:  path.Join(parts[3:]...),
	}
	if rp.Repo == "" {
		return repoPath{}, fmt.Errorf("%q does not identify a repository", s)
	}
	return rp, nil
}

func (rp repoPath) Validate() error {
	err := validateChars("username", rp.User, UsernameAllowedChars, UsernameAllowedMiddleChars)
	if err != nil {
		return err
	}
	err = validateChars("repository name", rp.Repo, RepoNameAllowedChars, RepoNameAllowedMiddleChars)
	if err != nil {
		return err
	}
	err = validateChars("offset directory", rp.Dir, OffsetAllowedChars, OffsetAllowedChars)
	if err != nil {
		return err
	}
	return nil
}

func validateChars(what, s, allowed, allowedMiddle string) error {
	lastChar := len(s) - 1
nextchar:
	for i, c := range s {
		if i == 0 || i == lastChar {
			for _, ac := range allowed {
				if c == ac {
					continue nextchar
				}
			}
		} else {
			for _, ac := range allowedMiddle {
				if c == ac {
					continue nextchar
				}
			}
		}
		return fmt.Errorf("%s %q contains illegal character '%c'", what, s, c)
	}
	return nil
}
