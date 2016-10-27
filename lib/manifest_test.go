package sous

import (
	"fmt"
	"testing"
)

func TestManifest_Validate(t *testing.T) {
	m := &Manifest{}
	flaws := m.Validate()
	expectedNumFlaws := 1
	if len(flaws) != expectedNumFlaws {
		t.Fatalf("got %d flaws; want %d", len(flaws), expectedNumFlaws)
	}
	expectedFlawDesc := `manifest "" missing Kind`
	actualFlawDesc := fmt.Sprint(flaws[0])
	if actualFlawDesc != expectedFlawDesc {
		t.Errorf("got flaw desc %q; want %q", actualFlawDesc, expectedFlawDesc)
	}
	if err := flaws[0].Repair(); err != nil {
		t.Fatal(err)
	}
	expectedKind := ManifestKindService
	actualKind := m.Kind
	if actualKind != expectedKind {
		t.Errorf("got Kind %q; want %q", actualKind, expectedKind)
	}
}
