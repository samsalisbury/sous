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
		named[name] = func(ds sous.Deployments) (sous.Deployments, error) {
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

func (f *DeploymentAttributeFilters) apply(ds sous.Deployments) (sous.Deployments, error) {
	var err error
	for _, filter := range f.filters {
		ds, err = filter(ds)
		if err != nil {
			return ds, err
		}
	}
	return ds, nil
}

type deployFilter func(sous.Deployments, bool) (sous.Deployments, error)
type boundDeployFilter func(sous.Deployments) (sous.Deployments, error)

func simpleFilter(p func(*sous.Deployment) bool) deployFilter {
	return func(ds sous.Deployments, which bool) (sous.Deployments, error) {
		return ds.Filter(func(d *sous.Deployment) bool {
			return p(d) == which
		}), nil
	}
}

func parallelFilter(maxConcurrent int, p func(*sous.Deployment) (bool, error)) deployFilter {
	return func(deployments sous.Deployments, which bool) (sous.Deployments, error) {
		if maxConcurrent < 1 {
			return deployments, fmt.Errorf("maxConcurrent < 1 not allowed")
		}
		// NOTE: We take snapshot here so that len cannot change. Deployments is
		// a concurrent map, so we have to assume len can change at any time.
		ds := deployments.Snapshot()
		// We take advantage of filtered being a concurrent map, writing to it
		// willy nilly from the goroutines we start in the loop below.
		filtered := sous.NewDeployments()

		wg := sync.WaitGroup{}
		wg.Add(len(ds))

		errs := make(chan error, len(ds))
		pool := make(chan struct{}, MaxConcurrentArtifactQueries)
		for i := 0; i < MaxConcurrentArtifactQueries; i++ {
			pool <- struct{}{}
		}

		for _, d := range ds {
			d := d
			<-pool
			go func() {
				defer wg.Done()
				defer func() { pool <- struct{}{} }()
				match, err := p(d)
				if err != nil {
					errs <- err
					return
				}
				if match != which {
					return
				}
				// This .Add call is safe because filtered is a concurrent map.
				filtered.Add(d)
			}()
		}
		wg.Wait()
		close(errs)

		for err := range errs {
			log.Println(err)
		}
		return filtered, nil
	}
}
