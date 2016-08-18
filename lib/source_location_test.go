package sous

import "testing"

var parseSourceLocationTests = map[string]SourceLocation{
	"github.com/opentable/sous,": {
		Repo: "github.com/opentable/sous",
	},
	":github.com/opentable/sous": {
		Repo: "github.com/opentable/sous",
	},
	",github.com/opentable/sous,util": {
		Repo: "github.com/opentable/sous",
		Dir:  "util",
	},
	":github.com/opentable/sous:util": {
		Repo: "github.com/opentable/sous",
		Dir:  "util",
	},
	"git+ssh://github.com/opentable/sous,sous": {
		Repo: "git+ssh://github.com/opentable/sous",
		Dir:  "sous",
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
