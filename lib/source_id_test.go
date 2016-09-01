package sous

import (
	"testing"

	"github.com/samsalisbury/semv"
)

var parseSourceIDTests = map[string]SourceID{
	"github.com/opentable/sous,1,": {
		SourceLocation: SourceLocation{
			Repo: "github.com/opentable/sous",
		},
		Version: semv.MustParse("1"),
	},
	":github.com/opentable/sous:1:": {
		SourceLocation: SourceLocation{
			Repo: "github.com/opentable/sous",
		},
		Version: semv.MustParse("1"),
	},
	"github.com/opentable/sous,1": {
		SourceLocation: SourceLocation{
			Repo: "github.com/opentable/sous",
		},
		Version: semv.MustParse("1"),
	},
	",github.com/opentable/sous,1,util": {
		SourceLocation: SourceLocation{
			Repo: "github.com/opentable/sous",
			Dir:  "util",
		},
		Version: semv.MustParse("1"),
	},
	"github.com/opentable/sous,1,util": {
		SourceLocation: SourceLocation{
			Repo: "github.com/opentable/sous",
			Dir:  "util",
		},
		Version: semv.MustParse("1"),
	},
	":github.com/opentable/sous:1:util": {
		SourceLocation: SourceLocation{
			Repo: "github.com/opentable/sous",
			Dir:  "util",
		},
		Version: semv.MustParse("1"),
	},
	"git+ssh://github.com/opentable/sous,1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3,sous": {
		SourceLocation: SourceLocation{
			Repo: "git+ssh://github.com/opentable/sous",
			Dir:  "sous",
		},
		Version: semv.MustParse("1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3"),
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
