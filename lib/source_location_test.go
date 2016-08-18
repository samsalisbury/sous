package sous

import "testing"

var parseSourceLocationTests = map[string]SourceLocation{
	"github.com/opentable/sous,": {
		RepoURL: "github.com/opentable/sous",
	},
	":github.com/opentable/sous": {
		RepoURL: "github.com/opentable/sous",
	},
	",github.com/opentable/sous,util": {
		RepoURL:    "github.com/opentable/sous",
		RepoOffset: "util",
	},
	":github.com/opentable/sous:util": {
		RepoURL:    "github.com/opentable/sous",
		RepoOffset: "util",
	},
	"git+ssh://github.com/opentable/sous,sous": {
		RepoURL:    "git+ssh://github.com/opentable/sous",
		RepoOffset: "sous",
	},
}

func TestParseSourceLocation(t *testing.T) {
	for in, expected := range parseSourceLocationTests {
		actual, err := ParseSourceLocation(in)
		if err != nil {
			t.Error(err)
			continue
		}
		if actual != expected {
			t.Errorf("got %v; want %v", actual, expected)
		}
	}
}

func TestMustParseSourceLocation(t *testing.T) {
	for in, expected := range parseSourceLocationTests {
		actual := MustParseSourceLocation(in)
		if actual != expected {
			t.Errorf("got %v; want %v", actual, expected)
		}
	}
}
