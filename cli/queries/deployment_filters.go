package queries

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"sync"

	sous "github.com/opentable/sous/lib"
)

// MaxConcurrentArtifactQueries is the max number of concurrent artifact
// queries. 100 is a conservative value to ensure we don't run out of file
// descriptors locally.
// NOTE: If users complain this is too slow we could make this configurable
//       by env var, or perhaps lookup the max file descriptors via ulimit -n...
const MaxConcurrentArtifactQueries = 100

// DeploymentFilters is the argument that determines which deployments are
// returned by a query.
type DeploymentFilters struct {
	AttributeFilters DeploymentAttributeFilters
}

// DeploymentAttributeFilters filters deployments based on their attributes.
type DeploymentAttributeFilters struct {
	filters []boundDeployFilter
	flagMap map[string]*string
}

// AddFlags adds the available filters from q as flags to fs.
func (f *DeploymentAttributeFilters) AddFlags(q *Deployment, fs *flag.FlagSet) {
	f.flagMap = map[string]*string{}
	for _, n := range q.availableFilterNames() {
		f.flagMap[n] = new(string)
		help := fmt.Sprintf("filter based on %s (true|false|<empty string>)", n)
		fs.StringVar(f.flagMap[n], n, "", help)
	}
}

// UnpackFlags should be called after flag.Parse and sets the filters up
// accordingly, overwriting any currently setup filters.
func (f *DeploymentAttributeFilters) UnpackFlags(q *Deployment) error {
	named := map[string]boundDeployFilter{}
	for name, val := range f.flagMap {
		if val == nil || *val == "" {
			continue
		}
		v, err := strconv.ParseBool(*val)
		if err != nil {
			return fmt.Errorf("value %q for flag %s not valid (want true or false)",
				val, name)
		}
		f, err := q.getFilter(name)
		if err != nil {
			return err
		}
		named[name] = func(ds sous.Deployments) sous.Deployments {
			return f(ds, v)
		}
	}
	f.filters = nil
	for _, name := range filterOrder {
		if filter, ok := named[name]; ok {
			f.filters = append(f.filters, filter)
		}
	}
	return nil
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
	pool := make(chan struct{}, MaxConcurrentArtifactQueries)
	for i := 0; i < MaxConcurrentArtifactQueries; i++ {
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
