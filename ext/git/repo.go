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
	Remote struct {
		Name, PushURL, FetchURL string
	}
	Remotes map[string]Remote
)

func (rs Remotes) AddFetch(name, url string) {
	r := rs.ensureExists(name)
	r.FetchURL = url
	rs[name] = r
}

func (rs Remotes) AddPush(name, url string) {
	r := rs.ensureExists(name)
	r.PushURL = url
	rs[name] = r
}

func (rs Remotes) ensureExists(name string) Remote {
	if _, ok := rs[name]; !ok {
		rs[name] = Remote{}
	}
	return rs[name]
}

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
	var (
		revision, branch, nearestTagName,
		repoRelativeDir string
		files, modifiedFiles, newFiles []string
		allTags                        []sous.Tag
		remotes                        Remotes
	)
	c := r.Client
	if err := parallel.Do(
		func(err *error) { branch, *err = c.CurrentBranch() },
		func(err *error) { revision, *err = c.Revision() },
		func(err *error) {
			repoRelativeDir, *err = filepath.Rel(r.Root, r.Client.Sh.Dir)
			if repoRelativeDir == "." {
				repoRelativeDir = ""
			}
		},
		func(err *error) {
			allTags, *err = r.Client.ListTags()
			if err != nil || len(allTags) == 0 {
				return
			}
			nearestTagName, *err = c.NearestTag()
			if err != nil {
				return
			}
			//nearestTagRevision, *err = c.RevisionAt(nearestTagName)
		},
		func(err *error) { files, *err = c.ListFiles() },
		func(err *error) { modifiedFiles, *err = c.ModifiedFiles() },
		func(err *error) { newFiles, *err = c.NewFiles() },
		func(err *error) { remotes, *err = c.ListRemotes() },
	); err != nil {
		return nil, err
	}

	var (
		primaryRemote    Remote
		primaryRemoteURL string
		ok               = false
	)

	primaryRemote, ok = remotes["upstream"]
	if !ok {
		primaryRemote, ok = remotes["origin"]
	}
	if ok {
		primaryRemoteURL = primaryRemote.FetchURL
	}

	return &sous.SourceContext{
		RootDir:                  r.Root,
		OffsetDir:                repoRelativeDir,
		Branch:                   branch,
		Revision:                 revision,
		Files:                    files,
		ModifiedFiles:            modifiedFiles,
		NewFiles:                 newFiles,
		Tags:                     allTags,
		NearestTagName:           nearestTagName,
		PossiblePrimaryRemoteURL: primaryRemoteURL,
		DirtyWorkingTree:         len(modifiedFiles)+len(newFiles) != 0,
	}, nil
}
