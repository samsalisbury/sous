package queries

import (
	"log"
	"sync"

	sous "github.com/opentable/sous/lib"
)

// DeploymentFilters is the argument that determines which deployments are
// returned by a query.
type DeploymentFilters struct {
	AttributeFilters *DeploymentAttributeFilters
}

// DeploymentAttributeFilters filters deployments based on their attributes.
type DeploymentAttributeFilters struct {
	filters []boundDeployFilter
}

func (f *DeploymentAttributeFilters) apply(ds sous.Deployments) sous.Deployments {
	for _, f := range f.filters {
		ds = f(ds)
	}
	return ds
}

type deployFilter func(sous.Deployments, bool) sous.Deployments
type boundDeployFilter func(sous.Deployments) sous.Deployments

func simpleFilter(p func(*sous.Deployment) bool) deployFilter {
	return func(ds sous.Deployments, which bool) sous.Deployments {
		return ds.Filter(func(d *sous.Deployment) bool {
			return p(d) == which
		})
	}
}

func newHasImageFilter(aq ArtifactQuery) hasImageFilter {
	return hasImageFilter{check: aq.Exists}
}

type hasImageFilter struct {
	check func(sous.SourceID) (bool, error)
}

func (f hasImageFilter) hasImage(deployments sous.Deployments, which bool) sous.Deployments {
	filtered := sous.NewDeployments()
	wg := sync.WaitGroup{}

	ds := deployments.Snapshot()

	wg.Add(len(ds))
	errs := make(chan error, len(ds))
	maxConcurrent := 10
	pool := make(chan struct{}, maxConcurrent)
	for i := 0; i < maxConcurrent; i++ {
		pool <- struct{}{}
	}

	for _, d := range ds {
		d := d
		go func() {
			defer wg.Done()
			<-pool
			defer func() { pool <- struct{}{} }()
			exists, err := f.check(d.SourceID)
			if err != nil {
				errs <- err
				return
			}
			if exists == which {
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
