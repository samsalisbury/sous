package sous

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

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

// Diff returns a true and a list of differences if ms and other are different.
// Otherwise it returns (false, nil).
func (ms Manifests) Diff(other Manifests) (bool, []string) {
	o := other.Snapshot()
	p := ms.Snapshot()
	var diffs []string
	diff := func(format string, a ...interface{}) { diffs = append(diffs, fmt.Sprintf(format, a...)) }
	for mid, m := range p {
		n, ok := o[mid]
		if !ok {
			diff("missing manifest %q", mid)
			continue
		}
		_, diffs := m.Diff(n)
		for _, d := range diffs {
			diff("manifest %q: %s", mid, d)
		}
	}
	for mid := range o {
		if _, ok := p[mid]; !ok {
			diff("extra manifest %q", mid)
		}
	}
	return len(diffs) != 0, diffs
}

func (ms Manifests) String() string {
	var mids []string
	for _, mid := range ms.Keys() {
		mids = append(mids, mid.String())
	}
	return fmt.Sprintf("Manifests(%s)", strings.Join(mids, ", "))
}

// Flavors returns all the flavors of manifests in this set of manifests.
func (ms Manifests) Flavors() []string {
	flavors := map[string]struct{}{}
	for _, mid := range ms.Keys() {
		flavors[mid.Flavor] = struct{}{}
	}
	var fs []string
	for f := range flavors {
		fs = append(fs, f)
	}
	return fs
}
