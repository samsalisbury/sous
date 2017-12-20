package sous

import "testing"

func TestSingleRectification(t *testing.T) {

	dp := DeployablePair{}

	sr := NewSingleRectification(dp)

	d := &DummyDeployer{}

	r := sr.Resolve(d)

	t.Log(r)

}
