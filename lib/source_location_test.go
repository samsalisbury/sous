package sous

import "testing"

var parseSourceLocationTests = map[string]SourceLocation{
	"github.com/opentable/sous,1,": {
		RepoURL: "github.com/opentable/sous",
	},
	":github.com/opentable/sous:1:": {
		RepoURL: "github.com/opentable/sous",
	},
	"github.com/opentable/sous,1": {
		RepoURL: "github.com/opentable/sous",
	},
	",github.com/opentable/sous,1,util": {
		RepoURL:    "github.com/opentable/sous",
		RepoOffset: "util",
	},
	"github.com/opentable/sous,1,util": {
		RepoURL:    "github.com/opentable/sous",
		RepoOffset: "util",
	},
	":github.com/opentable/sous:1:util": {
		RepoURL:    "github.com/opentable/sous",
		RepoOffset: "util",
	},
	"git+ssh://github.com/opentable/sous,1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3,sous": {
		RepoURL:    "git+ssh://github.com/opentable/sous",
		RepoOffset: "sous",
	},
}

func TestParseSourceVersion(t *testing.T) {
	for in, expected := range parseSourceIDTests {
		actual, err := ParseSourceID(in)
		if err != nil {
			t.Error(err)
			continue
		}
		if actual != expected {
			t.Errorf("got %v; want %v", actual, expected)
		}
	}
}
