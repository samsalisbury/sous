package sous

import (
	"fmt"
	"testing"
)

func TestManifests_Diff(t *testing.T) {

	manifest := func(id string, setFields ...func(m *Manifest)) *Manifest {
		m := &Manifest{}
		mid, err := ParseManifestID(id)
		if err != nil {
			panic(err)
		}
		m.SetID(mid)
		for _, f := range setFields {
			f(m)
		}
		return m
	}

	testCases := []struct {
		A, B  Manifests
		Diffs []string
	}{
		{
			NewManifests(),
			NewManifests(),
			nil,
		},
		{
			NewManifests(manifest("a")),
			NewManifests(manifest("b")),
			[]string{
				`missing manifest "a"`,
				`extra manifest "b"`,
			},
		},
		{
			NewManifests(manifest("a", func(m *Manifest) { m.Kind = ManifestKindWorker })),
			NewManifests(manifest("a", func(m *Manifest) { m.Kind = ManifestKindWorker })),
			nil,
		},
		{
			NewManifests(manifest("a", func(m *Manifest) { m.Kind = ManifestKindWorker })),
			NewManifests(manifest("a", func(m *Manifest) { m.Kind = ManifestKindOnDemand })),
			[]string{
				`manifest "a": kind; this: "worker"; other: "on-demand"`,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s.Diff(%s)", tc.A, tc.B), func(t *testing.T) {
			different, actual := tc.A.Diff(tc.B)
			expected := tc.Diffs
			if expected == nil && actual == nil {
				if different {
					t.Errorf("different == true; want false")
				}
				return
			}
			for i, expectedDiff := range expected {
				if len(actual) <= i {
					t.Errorf("got %d diffs; want %d", len(actual), len(expected))
					break
				}
				actualDiff := actual[i]
				if actualDiff != expectedDiff {
					t.Errorf("got diff %q; want %q", actualDiff, expectedDiff)
				}
			}
		})
	}
}
