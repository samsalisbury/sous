package cli

import (
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

func newSourceContextFunc(g GitSourceContext, f *SourceFlags) SourceContextFunc {
	c := g.SourceContext
	if c == nil {
		c = &sous.SourceContext{}
	}
	return func() (*sous.SourceContext, error) {
		sl, err := resolveSourceLocation(f, c)
		if err != nil {
			return nil, errors.Wrap(err, "resolving source location")
		}
		if sl != c.SourceLocation() {
			// TODO: Clone the repository, and use the cloned dir as source context.
			return nil, errors.Errorf("source location %q is not the same as the remote %q",
				sl, c.SourceLocation())
		}
		return c, nil
	}
}

func resolveSourceLocation(f *SourceFlags, c *sous.SourceContext) (sous.SourceLocation, error) {
	if c == nil {
		c = &sous.SourceContext{}
	}
	if f == nil {
		f = &SourceFlags{}
	}
	var repo, offset = c.PrimaryRemoteURL, c.OffsetDir
	if f.Repo != "" {
		repo = f.Repo
	}
	if f.Repo != "" {
		repo = f.Repo
		offset = ""
	}
	if f.Offset != "" {
		if f.Repo == "" {
			return sous.SourceLocation{}, errors.Errorf("you specified -offset but not -repo")
		}
		offset = f.Offset
	}
	if repo == "" {
		return sous.SourceLocation{}, errors.Errorf("no repo specified, please use -repo or run sous inside a git repo")
	}
	return sous.SourceLocation{
		RepoURL:    sous.RepoURL(repo),
		RepoOffset: sous.RepoOffset(offset),
	}, nil
}
