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

func TestSourceLocation_UnmarshalText(t *testing.T) {
	for in, expected := range parseSourceLocationTests {
		var actual SourceLocation
		if err := actual.UnmarshalText([]byte(in)); err != nil {
			t.Error(err)
			continue
		}
		if actual != expected {
			t.Errorf("%q got %#v; want %#v", in, actual, expected)
		}
	}
}

func TestSourceLocationShortName(t *testing.T) {
	expected := "project"
	sl := SourceLocation{
		Repo: "fake.tld/org/" + expected,
		Dir:  "down/here",
	}
	t.Logf("Generating ShortName for %#v", sl)
	shortName, err := sl.ShortName()
	if err != nil {
		t.Fatal(err)
	}
	if shortName != expected {
		t.Fatalf("Got:%s, expected:%s", shortName, expected)
	} else {
		t.Logf("Got %s, expected:%s", shortName, expected)
	}
}

func TestSourceLocationShortNameWithBadRepo(t *testing.T) {
	expected := "project"
	sl := SourceLocation{
		Repo: "NotaURL" + expected,
		Dir:  "down/here",
	}
	t.Logf("Generating ShortName for %#v", sl)
	_, err := sl.ShortName()
	if err == nil {
		t.Fatal("An error was expected, but got nil.")
	} else {
		t.Logf("An error was correctly anticipated: %s", err)
	}
}
