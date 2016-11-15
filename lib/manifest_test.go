package sous

import (
	"fmt"
	"testing"

	"github.com/samsalisbury/semv"
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
	{
		OriginalManifest: &Manifest{
			Kind: ManifestKindService,
			Deployments: DeploySpecs{
				"some-cluster": DeploySpec{
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus": "1",
							// NOTE: Missing memory.
							"ports": "1",
						},
						NumInstances: 3,
					},
					Version: semv.MustParse("1"),
				},
			},
		},
		FixedManifest: &Manifest{
			Kind: ManifestKindService,
			Deployments: DeploySpecs{
				"some-cluster": DeploySpec{
					DeployConfig: DeployConfig{
						Resources: Resources{
							"cpus": "1",
							// NOTE: Memory repaired by setting to default.
							"memory": "100",
							"ports":  "1",
						},
						NumInstances: 3,
					},
					Version: semv.MustParse("1"),
				},
			},
		},
		FlawDesc: "Missing resource field: memory",
	},
	{
		// NOTE: This one is valid, hence no FlawDesc.
		OriginalManifest: &Manifest{
			Kind: ManifestKindService,
			Deployments: DeploySpecs{
				"Global": DeploySpec{
					DeployConfig: DeployConfig{
						// NOTE: These resources are inherited.
						Resources: Resources{
							"cpus":   "1",
							"memory": "256",
							"ports":  "1",
						},
					},
				},
				"some-cluster": DeploySpec{
					DeployConfig: DeployConfig{
						Resources: Resources{
						// NOTE: Empty; inherited from Global.
						},
						NumInstances: 3,
					},
					Version: semv.MustParse("1"),
				},
			},
		},
	},
}

func TestManifest_Validate(t *testing.T) {
	for _, test := range manifestTests {
		m := test.OriginalManifest
		flaws := m.Validate()
		expectedNumFlaws := 1
		if len(flaws) != expectedNumFlaws {
			for _, f := range flaws {
				t.Error(f)
			}
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
		expected := test.RepairError
		if test.RepairError == "" {
			if err != nil {
				t.Fatal(err)
			}
		} else if err == nil {
			t.Fatalf("got nil; want error %q", expected)
		} else {
			actual := err.Error()
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
