package sous

import "github.com/pkg/errors"

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
