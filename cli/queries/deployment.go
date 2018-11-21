package queries

import (
	"strconv"
	"strings"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// Deployment supports querying deployments.
type Deployment struct {
	StateManager  sous.StateManager
	ArtifactQuery ArtifactQuery
}

// DeploymentQueryResult is the result of the query.
type DeploymentQueryResult struct {
	// Deployments is the final query result.
	Deployments sous.Deployments
}

// ParseAttributeFilters parses deployment filters in the format:
//   <name>=<true|false> <name2>=<true|false>
// It returns a valid set of deployment filters.
func (q *Deployment) ParseAttributeFilters(s string) (*DeploymentAttributeFilters, error) {
	f, err := q.parseFilters(s)
	if err != nil {
		return nil, err
	}
	return &DeploymentAttributeFilters{filters: f}, nil
}

// Result returns all the deployments matched by f.
func (q *Deployment) Result(f DeploymentFilters) (DeploymentQueryResult, error) {
	s, err := q.StateManager.ReadState()
	if err != nil {
		return DeploymentQueryResult{Deployments: sous.NewDeployments()}, err
	}
	ds, err := s.Deployments()
	return DeploymentQueryResult{Deployments: f.AttributeFilters.apply(ds)}, err
}

// filterOrder dictates the order filters should be run in, if present.
var filterOrder = []string{"zeroinstances", "hasowners", "hasimage"}

func (q *Deployment) availableFilters() map[string]deployFilter {
	hif := newHasImageFilter(q.ArtifactQuery)
	return map[string]deployFilter{
		"hasimage": hif.hasImage,
		"zeroinstances": simpleFilter(func(d *sous.Deployment) bool {
			return d.NumInstances == 0
		}),
		"hasowners": simpleFilter(func(d *sous.Deployment) bool {
			return len(d.Owners) != 0
		}),
	}
}

func (q *Deployment) availableFilterNames() []string {
	var names []string
	for k := range q.availableFilters() {
		names = append(names, k)
	}
	return names
}

func (q *Deployment) badFilterNameError(attempted string) error {
	return cmdr.UsageErrorf("filter %q not recognised; pick one of: %s",
		attempted, strings.Join(q.availableFilterNames(), ", "))
}

func (q *Deployment) getFilter(name string) (deployFilter, error) {
	f, ok := q.availableFilters()[name]
	if !ok {
		return nil, q.badFilterNameError(name)
	}
	return f, nil
}

func (q *Deployment) parseFilters(s string) ([]boundDeployFilter, error) {
	if s == "" {
		return nil, nil
	}
	named := map[string]boundDeployFilter{}
	parts := strings.Fields(s)
	for _, p := range parts {
		kv := strings.Split(p, "=")
		if len(kv) != 2 {
			return nil, cmdr.UsageErrorf("filter %q not valid; format is <name>=(true|false)")
		}
		k, v := kv[0], kv[1]
		f, err := q.getFilter(k)
		if err != nil {
			return nil, err
		}
		tf, err := strconv.ParseBool(v)
		if err != nil {
			return nil, cmdr.UsageErrorf("filter %q accepts true or false, not %q", k, v)
		}
		named[k] = func(ds sous.Deployments) sous.Deployments {
			return f(ds, tf)
		}
	}
	var filters []boundDeployFilter
	for _, name := range filterOrder {
		if f, ok := named[name]; ok {
			filters = append(filters, f)
		}
	}
	return filters, nil
}
