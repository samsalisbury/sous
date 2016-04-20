package git

import (
	"fmt"
	"strings"

	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/util/shell"
	"github.com/samsalisbury/semv"
)

// Client is a git client that shells out to locally installed git. It requires
// that git is in the path by default, although you can override that by setting
// the Bin field to a path to a different Git.
// Client is used to perform git commands within a particular directory,
// determined by its shell.Sh instance.
type Client struct {
	// Sh is the *shell.Sh instance this client uses for all shell interaction.
	Sh *shell.Sh
	// Bin is the path to the git binary. This defaults to "git", therefore
	// relying that git is in the path.
	Bin string
	// Version is the version of git at Bin.
	Version semv.Version
}

// NewClient returns a git client, as long as `git --version` succeeds. The
// *shell.Sh is used for all commands. The client created by this func uses
// git in your path.
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

func (c *Client) stdoutLines(name string, args ...interface{}) ([]string, error) {
	args = append([]interface{}{name}, args...)
	return c.Sh.Cmd(c.Bin, args...).Lines()
}

func (c *Client) table(name string, args ...interface{}) ([][]string, error) {
	args = append([]interface{}{name}, args...)
	return c.Sh.Cmd(c.Bin, args...).Table()
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

// ListFiles lists all files that are tracked in the repo.
func (c *Client) ListFiles() ([]string, error) {
	return c.stdoutLines("ls-files")
}

func (c *Client) ModifiedFiles() ([]string, error) {
	return c.stdoutLines("ls-files", "--modified")
}

func (c *Client) NewFiles() ([]string, error) {
	return c.stdoutLines("ls-files", "--others", "--exclude-standard")
}

func (c *Client) ListTags() ([]sous.Tag, error) {
	lines, err := c.stdoutLines("log", "--date-order", "--tags", "--simplify-by-decoration", `--pretty=format:%H %aI %D`)
	if err != nil {
		return nil, err
	}
	// E.g. output...
	//1141dde555492ea0a6073a222b2607900d09b0b5 2015-10-02T12:12:01+01:00 tag: v0.0.1-alpha1, tag: v0.0.1-alpha
	tags := []sous.Tag{}
	for _, l := range lines {
		r := strings.SplitN(l, " ", 3)
		if len(r) != 3 || !strings.Contains(r[2], "tag: ") {
			continue
		}
		cleanTags := strings.Replace(r[2], "tag: ", "", -1)
		ts := strings.Split(cleanTags, ", ")
		for _, t := range ts {

			tags = append(tags, sous.Tag{Name: t, Revision: r[0]})
		}
	}
	return tags, nil
}

func (c *Client) ListRemotes() (Remotes, error) {
	t, err := c.table("remote", "-v")
	if err != nil {
		return nil, err
	}
	remotes := Remotes{}
	for _, row := range t {
		if len(row) != 3 {
			fmt.Println("%s", t)
			return nil, fmt.Errorf("git remote -v output in unexpected format, please report this error")
		}
		name := row[0]
		url := row[1]
		kind := row[2]
		switch kind {
		default:
			return nil, fmt.Errorf("git remote -v returned a remote URL type %s; expected (push) or (fetch)")
		case "(push)":
			remotes.AddPush(name, url)
		case "(fetch)":
			remotes.AddFetch(name, url)
		}
	}
	return remotes, nil
}

func (c *Client) NearestTag() (string, error) {
	return c.stdout("describe", "--tags", "--abbrev=0")
}

func (c *Client) CurrentBranch() (string, error) {
	return c.stdout("rev-parse", "--abbrev-ref", "HEAD")
}
