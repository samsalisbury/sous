package sous

import (
	"fmt"
	"testing"
)

var manifestTests = []struct {
	OriginalManifest, FixedManifest *Manifest
	FlawDesc, RepairError           string
}{
	{
		OriginalManifest: &Manifest{},
		FixedManifest:    &Manifest{Kind: ManifestKindService},
		FlawDesc:         `manifest "" missing Kind`},
	{
		OriginalManifest: &Manifest{Kind: "some invalid kind"},
		FixedManifest:    &Manifest{Kind: "some invalid kind"},
		FlawDesc:         `ManifestKind "some invalid kind" not valid`,
		RepairError:      "unable to repair invalid ManifestKind",
	},
}

func TestManifest_Validate(t *testing.T) {
	for _, test := range manifestTests {
		m := test.OriginalManifest
		flaws := m.Validate()
		expectedNumFlaws := 1
		if len(flaws) != expectedNumFlaws {
			t.Fatalf("got %d flaws; want %d", len(flaws), expectedNumFlaws)
		}
		if test.FlawDesc != "" {
			expectedFlawDesc := test.FlawDesc
			actualFlawDesc := fmt.Sprint(flaws[0])
			if actualFlawDesc != expectedFlawDesc {
				t.Errorf("got flaw desc %q; want %q", actualFlawDesc, expectedFlawDesc)
			}
		}
		err := flaws[0].Repair()
		if test.RepairError == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else {
			actual := err.Error()
			expected := test.RepairError
			if actual != expected {
				t.Errorf("got error %q; want %q", actual, expected)
			}
		}
		if test.FixedManifest != nil {
			different, differences := m.Diff(test.FixedManifest)
			if different {
				t.Errorf("repaired manifest not as expected: % #v", differences)
			}
		}
	}
}
