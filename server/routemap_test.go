package server

import "testing"

type pm map[string]string

func TestSousRoutes(t *testing.T) {
	test := func(er string, name string, kvs ...KV) {
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
		KV{"repo", "github.com/opentable/sous"},
		KV{"offset", "alt"},
		KV{"flavor", "sweet"},
	)
	test(
		"/manifest?offset=alt&repo=github.com%2Fopentable%2Fsous",

		"manifest",
		KV{"repo", "github.com/opentable/sous"},
		KV{"offset", "alt"},
	)
}
