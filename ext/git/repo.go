package git

import (
	"path/filepath"

	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/util/parallel"
)

type (
	// Repo is a git repository.
	Repo struct {
		// Root is the root dir of this repo
		Root string
		// Client is a git client inside this repo.
		Client *Client
	}
	// Context is a snapshot of data from this Git repository.
	Context struct {
		Revision        string
		NearestTag      Tag
		RepoRelativeDir string
	}
	Tag struct {
		Name, Revision string
	}
)

// NewRepo takes a client, which it expects to already be inside a repo
// directory. It returns an error if the client is not inside a repository
// or if it fails to determine that fact. Note that it can be anywhere in a
// repository, it doesn't need to be in the root.
func NewRepo(c *Client) (*Repo, error) {
	root, err := c.RepoRoot()
	if err != nil {
		return nil, err
	}
	return &Repo{
		Root:   root,
		Client: c,
	}, nil
}

// SourceContext gathers together a number of bits of information about the
// repository such as its current branch, revision, nearest tag, nearest semver
// tag, etc.
func (r *Repo) SourceContext() (*sous.SourceContext, error) {
	var revision, nearestTagName, nearestTagRevision, repoRelativeDir string
	var allTags []sous.Tag
	err := parallel.Do(
		func(err *error) { revision, *err = r.Client.Revision() },
		func(err *error) {
			repoRelativeDir, *err = filepath.Rel(r.Root, r.Client.Sh.Dir)
		},
		func(err *error) {
			allTags, *err = r.Client.ListTags()
			if err != nil || len(allTags) == 0 {
				return
			}
			nearestTagName, *err = r.Client.NearestTag()
			if err != nil {
				return
			}
			nearestTagRevision, *err = r.Client.RevisionAt(nearestTagName)
		},
	)
	if err != nil {
		return nil, err
	}
	return &sous.SourceContext{
		RootDir:   r.Root,
		OffsetDir: repoRelativeDir,
		Revision:  revision,
		Tags:      allTags,
	}, nil
}
