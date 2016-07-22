package hy_test

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
		Dict: map[string]string{
			"a": "α",
			"b": "β",
		},
		Things: map[string]TestThing{
			"thingio": TestThing{Name: "Thingio"},
			"thingy":  TestThing{Name: "Thingy"},
		},
		Widgets: map[string]TestWidget{
			"some/random/dir/widge": TestWidget{Name: "Widge"},
			"some/other/tree/wodge": TestWidget{Name: "Wodge"},
		},
	}
	outDir := "./test/output"
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
