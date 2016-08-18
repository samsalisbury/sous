package sous

import (
	"testing"

	"github.com/samsalisbury/semv"
)

var parseSourceIDTests = map[string]SourceID{
	"github.com/opentable/sous,1,": {
		RepoURL: "github.com/opentable/sous",
		Version: semv.MustParse("1"),
	},
	":github.com/opentable/sous:1:": {
		RepoURL: "github.com/opentable/sous",
		Version: semv.MustParse("1"),
	},
	"github.com/opentable/sous,1": {
		RepoURL: "github.com/opentable/sous",
		Version: semv.MustParse("1"),
	},
	",github.com/opentable/sous,1,util": {
		RepoURL:    "github.com/opentable/sous",
		Version:    semv.MustParse("1"),
		RepoOffset: "util",
	},
	"github.com/opentable/sous,1,util": {
		RepoURL:    "github.com/opentable/sous",
		Version:    semv.MustParse("1"),
		RepoOffset: "util",
	},
	":github.com/opentable/sous:1:util": {
		RepoURL:    "github.com/opentable/sous",
		Version:    semv.MustParse("1"),
		RepoOffset: "util",
	},
	"git+ssh://github.com/opentable/sous,1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3,sous": {
		RepoURL:    "git+ssh://github.com/opentable/sous",
		Version:    semv.MustParse("1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3"),
		RepoOffset: "sous",
	},
}

func TestParseSourceID(t *testing.T) {
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
