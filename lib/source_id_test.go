package sous

import (
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestParseSourceID(t *testing.T) {
	assert := assert.New(t)
	mustParse := func(str string) SourceID {
		sv, err := ParseSourceID(str)
		if err != nil {
			t.Errorf("unexpected error %q while parsing %q", err, str)
		}
		return sv
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
