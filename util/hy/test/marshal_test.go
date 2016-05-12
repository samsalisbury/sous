package test

import (
	"os"
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
	outDir := "./test_output"
	marshaller := hy.NewMarshaller(yaml.Marshal)
	if err := marshaller.Marshal(outDir, base); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	f, err := os.Stat(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if !f.IsDir() {
		t.Fatalf("expected %s to be a directory", outDir)
	}
}
