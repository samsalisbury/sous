package sous

import (
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
func (ds Deployments) EmptyReceiver() Comparable {
	return NewDeployments()
}

// VariancesFrom implements Comparable on Deployments
func (ds Deployments) VariancesFrom(other Comparable) Variances {
	switch ods := other.(type) {
	default:
		return Variances{"not a list of Deployments"}
	case *Deployments:
		vs := Variances{}
		if ods.Len() != ds.Len() {
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
