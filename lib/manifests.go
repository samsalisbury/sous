package sous

import "github.com/pkg/errors"

// Only returns the single Manifest in a Manifests
//   XXX consider for inclusion in CMap
func (ms *Manifests) Only() (*Manifest, error) {
	switch ms.Len() {
	default:
		return nil, errors.Errorf("multiple manifests returned from Only:\n%#v", ms)
	case 0:

		return nil, nil
	case 1:
		p, got := ms.Get(ms.Keys()[0])
		if !got {
			panic("Non-empty Manifests returned no value for a reported key")
		}
		return p, nil
	}
}
