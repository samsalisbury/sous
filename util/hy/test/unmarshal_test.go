package test

import (
	"testing"

	"github.com/opentable/sous/util/hy"
	"github.com/opentable/sous/util/yaml"
)

type (
	Base struct {
		Config Config `hy:"config.yaml"`
	}
	Config struct {
		Name string
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
}
