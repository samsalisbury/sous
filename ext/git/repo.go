package git

type (
	// Repo is a git repository.
	Repo struct {
		// Root is the root dir of this repo
		Root string
		// Client is a git client inside this repo.
		Client *Client
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
