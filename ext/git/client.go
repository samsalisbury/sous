package git

import (
	"fmt"

	"github.com/opentable/sous2/util/shell"
	"github.com/samsalisbury/semv"
)

// Client is a git client that shells out to locally installed git. It requires
// that git is in the path. Client is used to perform git commands within a
// particular directory, determined by its shell.Sh instance.
type Client struct {
	Sh      *shell.Sh
	Bin     string
	Version semv.Version
}

// NewClient returns a git client, as long as `git --version` succeeds. The
// *shell.Sh is used for all commands.
func NewClient(sh *shell.Sh) (*Client, error) {
	bin := "git"
	s, err := sh.Cmd(bin, "version").Stdout()
	if err != nil {
		return nil, err
	}
	v, err := semv.ParseAny(s)
	if err != nil {
		return nil, err
	}
	return &Client{sh, bin, v}, nil
}

// NewClientInVersionRange is similar to NewClient, but returns
// nil and and error if the version of the installed git client
// is not in the specified range.
func NewClientInVersionRange(sh *shell.Sh, r semv.Range) (*Client, error) {
	c, err := NewClient(sh)
	if err != nil {
		return nil, err
	}
	if !c.Version.Satisfies(r) {
		return nil, fmt.Errorf("git version %s does not satisfy range %s",
			c.Version, r)
	}
	return c, nil
}

func (c *Client) Clone() *Client {
	cp := *c
	cp.Sh = cp.Sh.Clone()
	return &cp
}

func (c *Client) OpenRepo(dirpath string) (*Repo, error) {
	sh := c.Sh.Clone()
	if err := sh.CD(dirpath); err != nil {
		return nil, err
	}
	return NewRepo(c.Clone())
}

func (c *Client) stdout(name string, args ...interface{}) (string, error) {
	args = append([]interface{}{name}, args...)
	return c.Sh.Stdout(c.Bin, args...)
}

func (c *Client) Dir() string {
	return c.Sh.Dir
}

func (c *Client) RevisionAt(ref string) (string, error) {
	return c.stdout("rev-parse", ref)
}

func (c *Client) Revision() (string, error) {
	return c.RevisionAt("HEAD")
}

func (c *Client) RepoRoot() (string, error) {
	return c.stdout("rev-parse", "--show-toplevel")
}

func (c *Client) NearestTag() (string, error) {
	return c.stdout("describe", "--tags", "--abbrev=0")
}
