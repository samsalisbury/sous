package queries

import (
	"log"
	"strconv"
	"strings"
	"sync"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// DeploymentQuery supports querying deployments.
type DeploymentQuery struct {
	StateManager  sous.StateManager
	ArtifactQuery ArtifactQuery
}

// DeploymentFilters is the argument that determines which deployments are
// returned by a query.
type DeploymentFilters struct {
	AttributeFilters *AttributeFilters
}

// AttributeFilters filters deployments based on their attributes.
type AttributeFilters struct {
	filters []boundFilter
}

// QueryResult is the result of the query.
type QueryResult struct {
	// Deployments is the final query result.
	Deployments sous.Deployments
}

// ParseAttributeFilters parses deployment filters in the format:
//   <name>=<true|false> <name2>=<true|false>
// It returns a valid set of deployment filters.
func (q *DeploymentQuery) ParseAttributeFilters(s string) (*AttributeFilters, error) {
	f, err := q.parseFilters(s)
	if err != nil {
		return nil, err
	}
	return &AttributeFilters{filters: f}, nil
}

// Result returns all the deployments matched by f.
func (q *DeploymentQuery) Result(f DeploymentFilters) (QueryResult, error) {
	s, err := q.StateManager.ReadState()
	if err != nil {
		return QueryResult{Deployments: sous.NewDeployments()}, err
	}
	ds, err := s.Deployments()
	return QueryResult{Deployments: f.AttributeFilters.apply(ds)}, err
}

type deployFilter func(sous.Deployments, bool) sous.Deployments
type boundFilter func(sous.Deployments) sous.Deployments

func simpleFilter(p func(*sous.Deployment) bool) deployFilter {
	return func(ds sous.Deployments, which bool) sous.Deployments {
		return ds.Filter(func(d *sous.Deployment) bool {
			return p(d) == which
		})
	}
}

func (q *DeploymentQuery) availableFilters() map[string]deployFilter {
	return map[string]deployFilter{
		"hasimage": q.hasImageFilter,
		"zeroinstances": simpleFilter(func(d *sous.Deployment) bool {
			return d.NumInstances == 0
		}),
		"hasowners": simpleFilter(func(d *sous.Deployment) bool {
			return len(d.Owners) != 0
		}),
	}
}

func (q *DeploymentQuery) availableFilterNames() []string {
	var names []string
	for k := range q.availableFilters() {
		names = append(names, k)
	}
	return names
}

func (q *DeploymentQuery) hasImageFilter(deployments sous.Deployments, which bool) sous.Deployments {
	filtered := sous.NewDeployments()
	wg := sync.WaitGroup{}

	ds := deployments.Snapshot()

	wg.Add(len(ds))
	errs := make(chan error, len(ds))

	for _, d := range ds {
		d := d
		go func() {
			defer wg.Done()
			a, err := q.ArtifactQuery.ByID(d.SourceID)
			if err != nil {
				errs <- err
				return
			}
			if a != nil == which {
				filtered.Add(d)
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		log.Println(err)
	}
	return filtered
}

func (q *DeploymentQuery) badFilterNameError(attempted string) error {
	return cmdr.UsageErrorf("filter %q not recognised; pick one of: %s",
		attempted, strings.Join(q.availableFilterNames(), ", "))
}

func (q *DeploymentQuery) getFilter(name string) (deployFilter, error) {
	f, ok := q.availableFilters()[name]
	if !ok {
		return nil, q.badFilterNameError(name)
	}
	return f, nil
}

func (q *DeploymentQuery) parseFilters(s string) ([]boundFilter, error) {
	var filters []boundFilter
	if s == "" {
		return nil, nil
	}
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
		filters = append(filters, func(ds sous.Deployments) sous.Deployments {
			return f(ds, tf)
		})
	}
	return filters, nil
}

func (f *AttributeFilters) apply(ds sous.Deployments) sous.Deployments {
	for _, f := range f.filters {
		ds = f(ds)
	}
	return ds
}
