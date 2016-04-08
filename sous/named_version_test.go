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

	nv, ok := name.(NamedVersion)
	if !ok {
		t.Errorf("Didn't parse a NamedVersion, instead: %T", nv)
		return
	}

	if string(nv.RepositoryName) != "git+ssh://github.com/opentable/sous" {
		t.Errorf("Bad repo: %q", nv.RepositoryName)
	}

	if nv.Path != "/sous/" {
		t.Errorf("Bad path: %q", nv.Path)
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
	mustParse := func(str string) NamedVersion {
		nv, err := ParseNamedVersion(str)
		if err != nil {
			t.Errorf("unexpected error %q while parsing %q", err, str)
		}
		return nv
	}

	assert.Equal(NamedVersion{"github.com/opentable/sous", semv.MustParse("1"), "/"}, mustParse(",github.com/opentable/sous,1,/"))
	assert.Equal(NamedVersion{"github.com/opentable/sous", semv.MustParse("1"), "/"}, mustParse("github.com/opentable/sous,1,/"))
	assert.Equal(NamedVersion{"github.com/opentable/sous", semv.MustParse("1"), "/"}, mustParse(":github.com/opentable/sous:1:/"))
}
