package hy_test

import (
	"testing"

	"github.com/opentable/sous/util/hy"
	"github.com/opentable/sous/util/yaml"
)

type (
	Base struct {
		Config  Config                `hy:"config.yaml"`
		Dict    map[string]string     `hy:"dict.yaml"`
		Things  map[string]TestThing  `hy:"things/"`
		Widgets map[string]TestWidget `hy:"widgets/**"`
	}
	TestThing struct {
		Name, Desc string
	}
	Config     TestThing
	TestWidget TestThing
)

func _TestUnmarshal_GoodData(t *testing.T) {
	b := Base{}
	u := hy.NewUnmarshaler(yaml.Unmarshal)
	if err := u.Unmarshal("./test/data", &b); err != nil {
		t.Fatal(err)
	}
	if b.Config.Name != "Dave" {
		t.Errorf("Config.Name was %q; want %q", b.Config.Name, "Dave")
	}
	if len(b.Things) == 0 {
		t.Errorf("Things had length 0")
	}
	thing, ok := b.Things["thing1"]
	if !ok {
		t.Fatalf("Things[thing1] not set; got %v", b.Things)
	}
	if thing.Name != "Thing One" {
		t.Errorf("Thing[thing1].Name was %s; want Thing One", thing.Name)
	}
	if len(b.Widgets) == 0 {
		t.Fatal("Widgets had length zero")
	}
	widget, ok := b.Widgets["wodgets/pingu/widget1"]
	if !ok {
		t.Fatalf("Widgets[wodgets/pingu/widget1] not set")
	}
	if widget.Name != "Pingu" {
		t.Errorf("Widgets[wodgets/pingu/widget1] = %s; want Pingu", widget.Name)
	}
}
