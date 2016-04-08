package sous

import (
	"testing"

	"github.com/samsalisbury/semv"
)

func TestParseNamedVersion(t *testing.T) {
	inExpectedOut := map[string]NamedVersion{
		",github.com/opentable/sous,1,/": NamedVersion{
			"github.com/opentable/sous",
			semv.MustParse("1"),
			"/",
		},
		"github.com/opentable/sous,1,/": NamedVersion{
			"github.com/opentable/sous",
			semv.MustParse("1"),
			"/",
		},
		":github.com/opentable/sous:1:/": NamedVersion{
			"github.com/opentable/sous",
			semv.MustParse("1"),
			"/",
		},
	}

	for input, expected := range inExpectedOut {
		actual, err := ParseNamedVersion(input)
		if err != nil {
			t.Errorf("unexpected error %q while parsing %q", err, input)
			continue
		}
		if actual != expected {
			t.Errorf("got % +v parsing %q; want %q", actual, input, expected)
		}
	}
}
