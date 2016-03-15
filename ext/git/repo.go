package git

import "sync"

type Repo struct {
	// Root is the root dir of this repo
	Root string
	// Client is a git client inside this repo.
	Client *Client
}

type Context struct {
	Revision        string
	NearestTag      Tag
	RepoRelativeDir string
}

type Tag struct {
	Name, Revision string
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

// Context gathers together a number of bits of information about the repository
// such as its current branch, revision, nearest tag, nearest semver tag, etc.
func (r *Repo) Context() (*Context, error) {
	var revision, nearestTagName, nearestTagRevision, repoRelativeDir string
	if err := DoAll(
		//func(err *error) { revision, *err = r.Client.Revision() },
		func(err *error) {
			//repoRelativeDir, *err = filepath.Rel(r.Root, r.Client.Sh.Dir)
		},
		func(err *error) {
			//nearestTagName, *err = r.Client.NearestTag()
			if *err != nil {
				return
			}
			//nearestTagRevision, *err = r.Client.RevisionAt(nearestTagName)
		},
	); err != nil {
		return nil, err
	}
	return &Context{
		RepoRelativeDir: repoRelativeDir,
		Revision:        revision,
		NearestTag: Tag{
			Name:     nearestTagName,
			Revision: nearestTagRevision,
		},
	}, nil
}

func DoAll(fs ...func(*error)) error {
	wg := sync.WaitGroup{}
	wg.Add(len(fs))
	errs := make(chan error)
	go func() { wg.Wait(); close(errs) }()
	for _, f := range fs {
		go func() {
			var err error
			f(&err)
			if err != nil {
				errs <- err
			}
			wg.Done()
		}()
	}
	return <-errs
}
