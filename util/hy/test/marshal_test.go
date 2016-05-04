package test

import (
	"testing"

	"github.com/opentable/sous/util/hy"
	"github.com/opentable/sous/util/yaml"
)

func TestMarshal_GoodData(t *testing.T) {
	base := &Base{
		Config: Config{
			Name: "Config name",
		},
		Things: map[string]Thing{
			"thingio": Thing{Name: "Thingio"},
			"thingy":  Thing{Name: "Thingy"},
		},
		Widgets: map[string]Widget{
			"some/random/dir/widge": Widget{Name: "Widge"},
			"some/other/tree/wodge": Widget{Name: "Wodge"},
		},
	}
	marshaller := hy.NewMarshaller(yaml.Marshal)
	if err := marshaller.Marshal("./test_output", base); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
}
