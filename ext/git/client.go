package git

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/opentable/sous/lib"
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

	sh.Cmd(bin, "config", "-l").Stdout()

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

// CloneClient produces a clone of the client.
func (c *Client) CloneClient() *Client {
	cp := *c
	cp.Sh = cp.Sh.Clone().(*shell.Sh)
	return &cp
}

// CloneRepo clones repo into localPath.
func (c *Client) CloneRepo(repo, localPath string) error {
	_, err := c.stdout("clone", repo, localPath)
	return err
}

// OpenRepo opens a repo.
func (c *Client) OpenRepo(dirpath string) (*Repo, error) {
	sh := c.Sh.Clone()
	if err := sh.CD(dirpath); err != nil {
		return nil, err
	}
	return NewRepo(c.CloneClient())
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

// Dir returns the current directory.
func (c *Client) Dir() string {
	return c.Sh.Dir()
}

// RevisionAt returns the revision at ref.
func (c *Client) RevisionAt(ref string) (string, error) {
	return c.stdout("rev-list", "-n", "1", ref)
}

// Revision returns the revision at HEAD.
func (c *Client) Revision() (string, error) {
	return c.RevisionAt("HEAD")
}

// RepoRoot returns the absolute root directory of the current repo.
func (c *Client) RepoRoot() (string, error) {
	return c.stdout("rev-parse", "--show-toplevel")
}

// ListFiles lists all files that are tracked in the repo.
func (c *Client) ListFiles() ([]string, error) {
	return c.stdoutLines("ls-files")
}

// ModifiedFiles returns the list of tracked, modified files.
func (c *Client) ModifiedFiles() ([]string, error) {
	return c.stdoutLines("ls-files", "--modified")
}

// NewFiles returns the list of untracked files.
func (c *Client) NewFiles() ([]string, error) {
	return c.stdoutLines("ls-files", "--others", "--exclude-standard")
}

// ListTags lists the tags in this repo.
func (c *Client) ListTags() ([]sous.Tag, error) {
	lines, err := c.stdoutLines("log", "--date-order", "--tags", "--simplify-by-decoration", `--pretty=format:%H %aI %D`)
	if err != nil {
		return nil, err
	}
	// E.g. output...
	//1141dde555492ea0a6073a222b2607900d09b0b5 2015-10-02T12:12:01+01:00 tag: v0.0.1-alpha1, tag: v0.0.1-alpha
	tags := c.parseTags(lines)
	return tags, nil
}

var tagRE = regexp.MustCompile(`tag: (.*)`)

func (c *Client) parseTags(lines []string) []sous.Tag {
	var tags []sous.Tag
	for _, l := range lines {
		r := strings.SplitN(l, " ", 3)
		if len(r) < 3 {
			continue
		}
		for _, tq := range strings.Split(r[2], ", ") {
			if m := tagRE.FindStringSubmatch(tq); m != nil {
				tags = append(tags, sous.Tag{Name: m[1], Revision: r[0]})
			}
		}
	}
	return tags
}

// ListUnpushedCommits returns a list of commit sha1s for commits that haven't been pushed to any remote
func (c *Client) ListUnpushedCommits() ([]string, error) {
	lines, err := c.stdoutLines("log", "--branches", "--not", "--remotes", "--pretty=%H")
	if err != nil {
		return nil, err
	}
	return lines, nil
}

// ListRemotes lists all configured remotes.
func (c *Client) ListRemotes() (Remotes, error) {
	t, err := c.table("remote", "-v")
	if err != nil {
		return nil, err
	}
	remotes := Remotes{}
	for _, row := range t {
		if len(row) != 3 {
			fmt.Printf("%s\n", t)
			return nil, fmt.Errorf("git remote -v output in unexpected format, please report this error")
		}
		name := row[0]
		url := row[1]
		kind := row[2]
		switch kind {
		default:
			return nil, fmt.Errorf("git remote -v returned a remote URL type %s; expected (push) or (fetch)", kind)
		case "(push)":
			remotes.AddPush(name, url)
		case "(fetch)":
			remotes.AddFetch(name, url)
		}
	}
	return remotes, nil
}

// NearestTag returns the nearest tag contained in the HEAD revision.
func (c *Client) NearestTag() (string, error) {
	return c.stdout("describe", "--tags", "--abbrev=0", "--always")
}

// CurrentBranch returns the currently checked out branch name.
func (c *Client) CurrentBranch() (string, error) {
	return c.stdout("rev-parse", "--abbrev-ref", "HEAD")
}
