package hy_test

import "github.com/opentable/sous/util/hy"

type Thing struct {
	Config  map[string]string `hy:"config.yaml"`
	Widgets map[string]Widget `hy:"widgets/"`
}

type Widget struct{ Colour string }

func Example() {
	t := Thing{
		Config: map[string]string{"some-key": "some-value"},
		Widgets: map[string]Widget{
			"blue":   {"Blue"},
			"orange": {"Orange"},
		},
	}
	hy.Marshal("some_dir", &t)
	// Output:
}
