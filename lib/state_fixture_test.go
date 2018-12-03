package sous

import (
	"testing"
)

func TestDefaultManifests(t *testing.T) {
	got := DefaultManifests(3, func(idx int, m *Manifest) {
		m.SetID(GenerateManifestID(idx, ManifestIDOpts{
			RepoFmt:   "github.com/user{{.Idx}}/repo{{.Idx}}",
			DirFmt:    "dir{{.Idx}}",
			FlavorFmt: "flavor{{.Idx}}",
		}))
	})
	wantLen := 3
	if got.Len() != wantLen {
		t.Fatalf("got %d manifests; want %d", got.Len(), wantLen)
	}
	t.Log(got)
}

func TestDefaultStateFixture(t *testing.T) {

	state := StateFixture(StateFixtureOpts{
		ClusterCount:  1,
		ManifestCount: 1,
		ManifestIDOpts: &ManifestIDOpts{
			RepoFmt:   "github.com/user{{.Idx}}/repo{{.Idx}}",
			DirFmt:    "dir{{.Idx}}",
			FlavorFmt: "flavor{{.Idx}}",
		},
	})
	wantLen := 1
	if state.Manifests.Len() != wantLen {
		t.Fatalf("got %d manifests; want %d", state.Manifests.Len(), wantLen)
	}
}
