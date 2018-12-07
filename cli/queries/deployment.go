package queries

import (
	"fmt"
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

// Result returns all the deployments matched by f.
func (q *Deployment) Result(f DeploymentFilters) (DeploymentQueryResult, error) {
	s, err := q.StateManager.ReadState()
	if err != nil {
		return DeploymentQueryResult{Deployments: sous.NewDeployments()}, err
	}
	ds, err := s.Deployments()
	if err != nil {
		return DeploymentQueryResult{},
			fmt.Errorf("getting initial deployments: %s", err)
	}
	ds, err = f.AttributeFilters.apply(ds)
	return DeploymentQueryResult{Deployments: ds}, err
}

// filterOrder dictates the order filters should be run in, if present.
var filterOrder = []string{"zeroinstances", "hasowners", "hasimage"}

func (q *Deployment) availableFilters() map[string]deployFilter {
	return map[string]deployFilter{
		"hasimage": parallelFilter(MaxConcurrentArtifactQueries,
			func(d *sous.Deployment) (bool, error) {
				return q.ArtifactQuery.Exists(d.SourceID)
			}),
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
