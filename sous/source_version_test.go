package sous

import (
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestParseName_SourceVersion(t *testing.T) {
	commas := "git+ssh://github.com/opentable/sous,1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3,/sous/"
	name, err := ParseGenName(commas)
	if err != nil {
		t.Error(err)
	}

	sv, ok := name.(SourceVersion)
	if !ok {
		t.Fatalf("Parsed a %T; want a  SourceVersion", sv)
	}

	if string(sv.RepoURL) != "git+ssh://github.com/opentable/sous" {
		t.Errorf("Bad repo: %q", sv.RepoURL)
	}

	if sv.RepoOffset != "/sous/" {
		t.Errorf("Bad path: %q", sv.RepoOffset)
	}

	if sv.Version.Major != 1 || sv.Version.Pre != "pre" {
		t.Errorf("Bad version: %q", sv.Version)
	}

	if sv.RevId() != "4f850e9030224f528cfdb085d558f8508d06a6d3" {
		t.Errorf("Bad revision: %q", sv.RevId())
	}
}

func TestParseSourceVersion(t *testing.T) {
	assert := assert.New(t)
	mustParse := func(str string) SourceVersion {
		sv, err := ParseSourceVersion(str)
		if err != nil {
			t.Errorf("unexpected error %q while parsing %q", err, str)
		}
		return sv
	}

	assert.Equal(SourceVersion{"git+ssh://github.com/opentable/sous", semv.MustParse("1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3"), "sous"},
		mustParse("git+ssh://github.com/opentable/sous,1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3,sous"))

	assert.Equal(SourceVersion{"github.com/opentable/sous", semv.MustParse("1"), "util"}, mustParse(",github.com/opentable/sous,1,util"))
	assert.Equal(SourceVersion{"github.com/opentable/sous", semv.MustParse("1"), "util"}, mustParse("github.com/opentable/sous,1,util"))
	assert.Equal(SourceVersion{"github.com/opentable/sous", semv.MustParse("1"), "util"}, mustParse(":github.com/opentable/sous:1:util"))

	assert.Equal(SourceVersion{"github.com/opentable/sous", semv.MustParse("1"), ""}, mustParse("github.com/opentable/sous,1,"))
	assert.Equal(SourceVersion{"github.com/opentable/sous", semv.MustParse("1"), ""}, mustParse(":github.com/opentable/sous:1:"))
	assert.Equal(SourceVersion{"github.com/opentable/sous", semv.MustParse("1"), ""}, mustParse("github.com/opentable/sous,1"))
}
