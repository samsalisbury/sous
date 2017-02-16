package server

import (
	"testing"

	"github.com/opentable/sous/util/restful"
)

type pm map[string]string

func TestSousRoutes(t *testing.T) {
	test := func(er string, name string, kvs ...restful.KV) {
		ar, err := SousRouteMap.PathFor(name, kvs...)
		if err != nil {
			t.Fatalf("Error getting a path: %#v", err)
		}
		if ar != er {
			t.Errorf("Route bad: expected %q got %q", er, ar)
		}

	}
	test(
		"/manifest?flavor=sweet&offset=alt&repo=github.com%2Fopentable%2Fsous",

		"manifest",
		restful.KV{"repo", "github.com/opentable/sous"},
		restful.KV{"offset", "alt"},
		restful.KV{"flavor", "sweet"},
	)
	test(
		"/manifest?offset=alt&repo=github.com%2Fopentable%2Fsous",

		"manifest",
		restful.KV{"repo", "github.com/opentable/sous"},
		restful.KV{"offset", "alt"},
	)
	test(
		"/status",
		"status",
	)
}
