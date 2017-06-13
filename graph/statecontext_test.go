package graph

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestNewStateWriteContext(t *testing.T) {

	mid := TargetManifestID{
		Source: sous.SourceLocation{
			Repo: "hai",
		},
	}

	u := sous.User{Name: "Some User"}

	actual := newStateWriteContext(mid, u)

	const expectedUser = "Some User"
	const expectedRepo = "hai"
	if actual.User.Name != expectedUser {
		t.Errorf("got %q; want %q", actual.User.Name, expectedUser)
	}
	actualRepo := actual.TargetManifestID.Source.Repo
	if actualRepo != expectedRepo {
		t.Errorf("got %q; want %q", actualRepo, expectedRepo)
	}
}

func TestNewStateReadContext(t *testing.T) {

	u := sous.User{Name: "Some User"}

	actual := newStateReadContext(u)

	const expectedUser = "Some User"
	if actual.User.Name != expectedUser {
		t.Errorf("got %q; want %q", actual.User.Name, expectedUser)
	}

	expectedTMID := sous.ManifestID{}
	if actual.TargetManifestID != expectedTMID {
		t.Errorf("got %q; want %q", actual.TargetManifestID, expectedTMID)
	}
}
