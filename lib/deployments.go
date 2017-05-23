package sous

import (
	"bytes"
	"fmt"

	"github.com/pkg/errors"
)

// Only returns the single Manifest in a Manifests
//   XXX consider for inclusion in CMap
//   c&p from manifests.go - absolutely consider for CMap
func (ds *Deployments) Only() (*Deployment, error) {
	switch ds.Len() {
	default:
		return nil, errors.Errorf("multiple deploys returned from Only:\n%#v", ds)
	case 0:
		return nil, nil
	case 1:
		p, got := ds.Get(ds.Keys()[0])
		if !got {
			panic("Non-empty Deployments returned no value for a reported key")
		}
		return p, nil
	}
}

// EmptyReceiver implements Comparable on Deployments
func (ds *Deployments) EmptyReceiver() Comparable {
	nds := NewDeployments()
	return &nds
}

// VariancesFrom implements Comparable on Deployments
func (ds *Deployments) VariancesFrom(other Comparable) Variances {

	switch ods := other.(type) {
	default:
		return Variances{"not a list of Deployments"}
	case *Deployments:
		vs := Variances{}
		if ds.Len() != ods.Len() {
			vs = append(vs, fmt.Sprintf("We have %d deployments, other has %d.", ds.Len(), ods.Len()))
		}
		for did, dep := range ds.Snapshot() {
			od, has := ods.Get(did)
			if !has {
				vs = append(vs, fmt.Sprintf("No deployment in other set for %v.", did))
			}
			_, diffs := dep.Diff(od)
			vs = append(vs, diffs...)
		}
		return vs
	}
}

// DeploymentIDSlice is a slice of DeploymentID, named so that it can be sort-able
type DeploymentIDSlice []DeploymentID

// Len implements sort.Interface on []DeploymentID
func (dids DeploymentIDSlice) Len() int {
	return len(dids)
}

// Less implements sort.Interface on []DeploymentID
func (dids DeploymentIDSlice) Less(i, j int) bool {
	return bytes.Compare(dids[i].Digest(), dids[j].Digest()) < 0
}

// Swap implements sort.Interface on []DeploymentID
func (dids DeploymentIDSlice) Swap(i, j int) {
	dids[i], dids[j] = dids[j], dids[i]
}
