package sous

import (
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestParseName_NamedVersion(t *testing.T) {
	commas := "git+ssh://github.com/opentable/sous,1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3,/sous/"
	name, err := ParseGenName(commas)
	if err != nil {
		t.Error(err)
	}

	nv, ok := name.(SourceID)
	if !ok {
		t.Fatalf("Parsed a %T; want a  NamedVersion", nv)
	}

	if string(nv.RepoURL) != "git+ssh://github.com/opentable/sous" {
		t.Errorf("Bad repo: %q", nv.RepoURL)
	}

	if nv.RepoOffset != "/sous/" {
		t.Errorf("Bad path: %q", nv.RepoOffset)
	}

	if nv.Version.Major != 1 || nv.Version.Pre != "pre" {
		t.Errorf("Bad version: %q", nv.Version)
	}

	if nv.RevId() != "4f850e9030224f528cfdb085d558f8508d06a6d3" {
		t.Errorf("Bad revision: %q", nv.RevId())
	}
}

func TestParseNamedVersion(t *testing.T) {
	assert := assert.New(t)
	mustParse := func(str string) SourceID {
		nv, err := ParseNamedVersion(str)
		if err != nil {
			t.Errorf("unexpected error %q while parsing %q", err, str)
		}
		return nv
	}

	assert.Equal(SourceID{"git+ssh://github.com/opentable/sous", semv.MustParse("1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3"), "sous"},
		mustParse("git+ssh://github.com/opentable/sous,1.0.0-pre+4f850e9030224f528cfdb085d558f8508d06a6d3,sous"))

	assert.Equal(SourceID{"github.com/opentable/sous", semv.MustParse("1"), "util"}, mustParse(",github.com/opentable/sous,1,util"))
	assert.Equal(SourceID{"github.com/opentable/sous", semv.MustParse("1"), "util"}, mustParse("github.com/opentable/sous,1,util"))
	assert.Equal(SourceID{"github.com/opentable/sous", semv.MustParse("1"), "util"}, mustParse(":github.com/opentable/sous:1:util"))

	assert.Equal(SourceID{"github.com/opentable/sous", semv.MustParse("1"), ""}, mustParse("github.com/opentable/sous,1,"))
	assert.Equal(SourceID{"github.com/opentable/sous", semv.MustParse("1"), ""}, mustParse(":github.com/opentable/sous:1:"))
	assert.Equal(SourceID{"github.com/opentable/sous", semv.MustParse("1"), ""}, mustParse("github.com/opentable/sous,1"))
}
