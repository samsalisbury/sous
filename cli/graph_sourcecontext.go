package cli

import (
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

func newSourceContextFunc(g GitSourceContext, f *SourceFlags) SourceContextFunc {
	c := g.SourceContext
	return func() (*sous.SourceContext, error) {
		sl, err := resolveSourceLocation(f, c)
		if err != nil {
			return nil, errors.Wrap(err, "resolving source location")
		}
		if sl != g.SourceLocation() {
			// TODO: Clone the repository, and use the cloned dir as source context.
			return nil, errors.Errorf("source location outside of current directory not yet supported")
		}
		return c, nil
	}
}

func resolveSourceLocation(f *SourceFlags, g *sous.SourceContext) (sous.SourceLocation, error) {
	if g == nil {
		g = &sous.SourceContext{}
	}
	if f == nil {
		f = &SourceFlags{}
	}
	var repo, offset = g.PrimaryRemoteURL, g.OffsetDir
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
