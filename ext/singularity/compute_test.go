package singularity

import (
	"github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"

	"strings"
	"testing"
)

func TestFlavoredComputeDeployIDLength(t *testing.T) {
	expectedVersion := "3.2.1"
	expectedFlavor := "frontdoor"

	v, err := semv.Parse(expectedVersion)
	if err != nil {
		t.Fatal(err)
	}

	d := &sous.Deployable{
		Deployment: &sous.Deployment{
			Flavor: expectedFlavor,
			SourceID: sous.SourceID{
				Version: v,
			},
		},
	}

	id := computeDeployID(d)
	expectedLength := 3
	lenTemplate := "Split deployID length: got %d elements, want %d elements."
	splitID := strings.Split(id, "-")
	length := len(splitID)
	if length != expectedLength {
		t.Fatalf(lenTemplate, length, expectedLength)
	} else {
		t.Logf(lenTemplate, length, expectedLength)
	}

}

func TestFlavoredComputeDeployVersion(t *testing.T) {
	expectedVersion := "3.2.1"
	expectedFlavor := "frontdoor"

	v, err := semv.Parse(expectedVersion)
	if err != nil {
		t.Fatal(err)
	}

	d := &sous.Deployable{
		Deployment: &sous.Deployment{
			Flavor: expectedFlavor,
			SourceID: sous.SourceID{
				Version: v,
			},
		},
	}

	id := computeDeployID(d)
	splitID := strings.Split(id, "-")
	versionTemplate := "Split deployID string:\"%s\" got version %s, want %s."
	if splitID[0] != expectedVersion {
		t.Fatalf(versionTemplate, id, splitID[0], expectedVersion)
	} else {
		t.Logf(versionTemplate, id, splitID[0], expectedVersion)
	}
}

func TestFlavoredComputeDeployFlavor(t *testing.T) {
	expectedFlavor := "frontdoor"
	v, err := semv.Parse("2.3.1")
	if err != nil {
		t.Fatal(err)
	}

	d := &sous.Deployable{
		Deployment: &sous.Deployment{
			Flavor: expectedFlavor,
			SourceID: sous.SourceID{
				Version: v,
			},
		},
	}

	id := computeDeployID(d)
	splitID := strings.Split(id, "-")
	flavorTemplate := "Split deployID string:\"%s\" got flavor %s, want %s."
	if splitID[1] != expectedFlavor {
		t.Fatalf(flavorTemplate, id, splitID[1], expectedFlavor)
	} else {
		t.Logf(flavorTemplate, id, splitID[1], expectedFlavor)
	}
}
