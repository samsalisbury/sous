package sous

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeployConfig_Validate_Repair(t *testing.T) {
	dc := DeployConfig{
		Volumes:   Volumes{nil, &Volume{}},
		Resources: make(Resources),
		Startup: Startup{

			SkipCheck: true,
		},
	}
	dc.Resources["cpus"] = "0.25"
	dc.Resources["memory"] = "356"
	dc.Resources["ports"] = "2"

	t.Log(dc.Startup)

	assert.Len(t, dc.Volumes, 2)
	flaws := dc.Validate()
	assert.Len(t, flaws, 1)
	fs, es := RepairAll(flaws)
	assert.Len(t, fs, 0)
	assert.Len(t, es, 0)
	assert.Len(t, dc.Volumes, 1)
}

// TODO: Add a more complete test for this Diff method.
// This one just tests the new SingularityRequestID field.
func TestDeployConfig_Diff_singularityRequestID(t *testing.T) {
	a := &DeployConfig{SingularityRequestID: "a"}
	b := DeployConfig{SingularityRequestID: "b"}
	different, diffs := a.Diff(b)
	if !different {
		t.Errorf("not different")
	}
	if len(diffs) != 1 {
		t.Fatalf("got %d diffs; want %d", len(diffs), 1)
	}
	got := diffs[0]
	want := `SingularityRequestID; this: "a"; other "b"`
	if got != want {
		t.Errorf("got diff %q; want %q", got, want)
	}
}
