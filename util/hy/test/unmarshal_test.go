package test

import (
	"testing"

	"github.com/opentable/sous/util/hy"
	"github.com/opentable/sous/util/yaml"
)

type (
	Base struct {
		Config Config           `hy:"config.yaml"`
		Things map[string]Thing `hy:"things/"`
	}
	Config struct {
		Name string
	}
	Thing struct {
		Name string
		Desc string
	}
)

func TestUnmarshal_GoodData(t *testing.T) {
	b := Base{}
	u := hy.NewUnmarshaler(yaml.Unmarshal)
	if err := u.Unmarshal("./data", &b); err != nil {
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
}
