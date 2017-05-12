package sous

import "testing"

// XXX doesn't test concentration of any kind..
func TestEmptyDiffConcentration(t *testing.T) {

	intended := NewDeployments()
	existing := NewDeployments()

	dc := intended.Diff(existing)
	ds := dc.collect()

	if len(ds.New) != 0 {
		t.Errorf("got %d new; want 0", len(ds.New))
	}
	if len(ds.Gone) != 0 {
		t.Errorf("got %d gone; want 0", len(ds.Gone))
	}
	if len(ds.Same) != 0 {
		t.Errorf("got %d same; want 0", len(ds.Same))
	}
	if len(ds.Changed) != 0 {
		t.Errorf("got %d changed; want 0", len(ds.Changed))

	}
}
