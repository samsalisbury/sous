package github

import (
	"fmt"
	"testing"

	"github.com/opentable/sous/lib"
)

var sourceLocationTests = []struct {
	String         string
	SourceLocation sous.SourceLocation
	Error          string
}{
	// wrong prefix
	{
		String: "",
		Error:  `"" does not begin with "github.com/"`,
	},
	{
		String: "hello",
		Error:  `"hello" does not begin with "github.com/"`,
	},
	{
		String: "github.com123",
		Error:  `"github.com123" does not begin with "github.com/"`,
	},
	{
		String: "http://github.com/some-user/some-repo",
		Error:  `"http://github.com/some-user/some-repo" does not begin with "github.com/"`,
	},
	{
		String: "git@github.com:some-user/some-repo",
		Error:  `"git@github.com:some-user/some-repo" does not begin with "github.com/"`,
	},
	{
		String: "ssh://git@github.com:some-user/some-repo",
		Error:  `"ssh://git@github.com:some-user/some-repo" does not begin with "github.com/"`,
	},
	{
		String: "git://github.com:some-user/some-repo",
		Error:  `"git://github.com:some-user/some-repo" does not begin with "github.com/"`,
	},

	// invalid characters in username
	{
		String: "github.com/some:user/some-repo",
		Error:  `username "some:user" contains illegal character ':'`,
	},
	{
		String: "github.com/some_user/some-repo",
		Error:  `username "some_user" contains illegal character '_'`,
	},
	{
		String: "github.com/some~user/some-repo",
		Error:  `username "some~user" contains illegal character '~'`,
	},

	// invalid characters in repository name
	{
		String: "github.com/some-user/some:repo",
		Error:  `repository name "some:repo" contains illegal character ':'`,
	},
	{
		String: "github.com/some-user/some~repo",
		Error:  `repository name "some~repo" contains illegal character '~'`,
	},

	// invalid characters in offset directory
	{
		String: "github.com/some-user/some-repo/offset:dir",
		Error:  `offset directory "offset:dir" contains illegal character ':'`,
	},
	{
		String: "github.com/some-user/some-repo,offset,dir",
		Error:  `offset directory "offset,dir" contains illegal character ','`,
	},
	{
		String: "github.com/some-user/some-repo/offset~dir",
		Error:  `offset directory "offset~dir" contains illegal character '~'`,
	},

	// does not identify a repository
	{
		String: "github.com/some-user",
		Error:  `"github.com/some-user" does not identify a repository`,
	},
	{
		String: "github.com/some-user/",
		Error:  `"github.com/some-user/" does not identify a repository`,
	},
	{
		String: "github.com/some-user,offset-dir",
		Error:  `"github.com/some-user" does not identify a repository`,
	},

	// success, repo only
	{
		String: "github.com/some-user/some-repo",
		SourceLocation: sous.SourceLocation{
			Repo: "github.com/some-user/some-repo",
		},
	},
	{
		String: "github.com/some-user/some_repo",
		SourceLocation: sous.SourceLocation{
			Repo: "github.com/some-user/some_repo",
		},
	},
	{
		String: "github.com/some-user/some.repo",
		SourceLocation: sous.SourceLocation{
			Repo: "github.com/some-user/some.repo",
		},
	},

	// standard offset notation using comma
	{
		String: "github.com/some-user/some-repo,some-offset",
		SourceLocation: sous.SourceLocation{
			Repo: "github.com/some-user/some-repo",
			Dir:  "some-offset",
		},
	},
	{
		String: "github.com/some-user/some-repo,some/offset",
		SourceLocation: sous.SourceLocation{
			Repo: "github.com/some-user/some-repo",
			Dir:  "some/offset",
		},
	},

	// permissive offset using slash, since we know github repos are
	// exactly of the form "github.com/<user>/<repo>"
	{
		String: "github.com/some-user/some-repo/some-offset",
		SourceLocation: sous.SourceLocation{
			Repo: "github.com/some-user/some-repo",
			Dir:  "some-offset",
		},
	},
	{
		String: "github.com/some-user/some-repo/some/offset",
		SourceLocation: sous.SourceLocation{
			Repo: "github.com/some-user/some-repo",
			Dir:  "some/offset",
		},
	},
}

func TestParseSourceLocation(t *testing.T) {
	for _, test := range sourceLocationTests {
		if err := checkParse(test.String, test.SourceLocation, test.Error); err != nil {
			t.Error(err)
		}
	}
}

func checkParse(input string, expected sous.SourceLocation, expectedErr string) error {
	actual, actualErr := ParseSourceLocation(input)
	if actualErr != nil && expectedErr == "" {
		return actualErr
	}
	if actualErr == nil && expectedErr != "" {
		return fmt.Errorf("%q got nil; want error:\n%#q", input, expectedErr)
	}
	if actualErr != nil && expectedErr != "" {
		actual, expected := actualErr.Error(), expectedErr
		if actual != expected {
			return fmt.Errorf("%q got error:\n%#q;\nwant:\n%#q", input, actual, expected)
		}
	}
	if actual != expected {
		return fmt.Errorf("%q got %#v; want %#v", input, actual, expected)
	}
	return nil
}
