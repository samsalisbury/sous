package sous

import (
	"testing"

	"github.com/samsalisbury/semv"
)

var parseSourceIDTests = map[string]SourceID{
	"github.com/opentable/sous,1,": {
		Location: SourceLocation{
			Repo: "github.com/opentable/sous",
		},
		Version: semv.MustParse("1"),
	},
	":github.com/opentable/sous:1:": {
		Location: SourceLocation{
			Repo: "github.com/opentable/sous",
		},
		Version: semv.MustParse("1"),
	},
	"github.com/opentable/sous,1": {
		Location: SourceLocation{
			Repo: "github.com/opentable/sous",
		},
		Version: semv.MustParse("1"),
	},
	",github.com/opentable/sous,1,util": {
		Location: SourceLocation{
			Repo: "github.com/opentable/sous",
			Dir:  "util",
		},
		Version: semv.MustParse("1"),
	},
	"github.com/opentable/sous,1,util": {
		Location: SourceLocation{
			Repo: "github.com/opentable/sous",
			Dir:  "util",
		},
		Version: semv.MustParse("1"),
	},
	":github.com/opentable/sous:1:util": {
		Location: SourceLocation{
			Repo: "github.com/opentable/sous",
			Dir:  "util",
		},
		Version: semv.MustParse("1"),
	},
	"git+ssh://github.com/opentable/sous,1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3,sous": {
		Location: SourceLocation{
			Repo: "git+ssh://github.com/opentable/sous",
			Dir:  "sous",
		},
		Version: semv.MustParse("1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3"),
	},
}

func TestParseSourceID_success(t *testing.T) {
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

func TestNewSourceID_success(t *testing.T) {
	for _, sid := range parseSourceIDTests {
		actual, err := NewSourceID(sid.Location.Repo, sid.Location.Dir, sid.Version.String())
		if err != nil {
			t.Error(err)
		}
		if actual != sid {
			t.Errorf("Got %+v; want %+v", actual, sid)
		}
	}
}

func TestNewSourceID_failure(t *testing.T) {
	sid, err := NewSourceID("", "", "not a version")
	expected := "unexpected character 'n' at position 0"
	if err == nil {
		t.Errorf("got nil; want error %q", expected)
	}
	if (sid != SourceID{}) {
		t.Errorf("got non-zero SourceID: %+v", sid)
	}
	actual := err.Error()
	if actual != expected {
		t.Errorf("got error %q; want %q", actual, expected)
	}
}

func TestMustNewSourceID_success(t *testing.T) {
	for _, sid := range parseSourceIDTests {
		actual := MustNewSourceID(sid.Location.Repo, sid.Location.Dir, sid.Version.String())
		if actual != sid {
			t.Errorf("Got %+v; want %+v", actual, sid)
		}
	}
}

func TestMustNewSourceID_failure(t *testing.T) {
	var actualErr error
	sid := func() SourceID {
		defer func() {
			if err := recover(); err != nil {
				actualErr = err.(error)
			}
		}()
		return MustNewSourceID("", "", "_")
	}()
	expected := "unexpected character '_' at position 0"
	if actualErr == nil {
		t.Errorf("got nil; want error %q", expected)
	}
	if (sid != SourceID{}) {
		t.Errorf("got non-zero SourceID: %+v", sid)
	}
	actual := actualErr.Error()
	if actual != expected {
		t.Errorf("got error %q; want %q", actual, expected)
	}
}

var sourceIDStringTests = map[string]SourceID{
	"github.com/opentable/sous,1": {
		Location: SourceLocation{
			Repo: "github.com/opentable/sous",
		},
		Version: semv.MustParse("1"),
	},
	"github.com/opentable/sous,1,util": {
		Location: SourceLocation{
			Repo: "github.com/opentable/sous",
			Dir:  "util",
		},
		Version: semv.MustParse("1"),
	},
	"git+ssh://github.com/opentable/sous,1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3,sous": {
		Location: SourceLocation{
			Repo: "git+ssh://github.com/opentable/sous",
			Dir:  "sous",
		},
		Version: semv.MustParse("1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3"),
	},
}

func TestSourceID_String(t *testing.T) {
	for expected, input := range sourceIDStringTests {
		actual := input.String()
		if actual != expected {
			t.Errorf("%+v got %q; want %q", input, actual, expected)
		}
	}
}

func TestSourceID_roundtrip_String_Parse(t *testing.T) {
	for _, sid := range sourceIDStringTests {
		intermediate := sid.String()
		expected := sid
		actual, err := ParseSourceID(intermediate)
		if err != nil {
			t.Error(err)
			continue
		}
		if actual != expected {
			t.Errorf("%+v round-tripped as %+v", actual, expected)
		}
	}
}
