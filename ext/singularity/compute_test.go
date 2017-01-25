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

func TestComputeDeployID_emptyFlavor(t *testing.T) {
	input := &sous.Deployable{
		Deployment: &sous.Deployment{
			Flavor: "",
			SourceID: sous.SourceID{
				Version: semv.MustParse("1.2.3"),
			},
		},
	}

	actual := computeDeployID(input)

	actualParts := strings.Split(actual, "-")
	if len(actualParts) != 2 {
		t.Fatalf("got %q; want a string with exactly one hyphen", actual)
	}
	if actualParts[0] != "1.2.3" {
		t.Fatalf("got %q; want a string beginning with 1.2.3", actual)
	}
	if len(actualParts[1]) != 32 {
		t.Fatalf("got %q want a string that ends with 32 hex digits", actual)
	}
}
