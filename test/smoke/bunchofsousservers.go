package smoke

import (
	"fmt"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

type bunchOfSousServers struct {
	BaseDir   string
	Count     int
	Instances []*sousServer
	Stop      func() error
}

func newBunchOfSousServers(t *testing.T, f *fixtureConfig) (*bunchOfSousServers, error) {

	binPath := sousBin

	state := f.InitialState

	count := len(state.Defs.Clusters)
	instances := make([]*sousServer, count)
	addrs := freePortAddrs("127.0.0.1", count)
	for i := 0; i < count; i++ {
		clusterName := state.Defs.Clusters.Names()[i]
		inst, err := makeInstance(t, f, binPath, i, clusterName, addrs[i])
		if err != nil {
			return nil, errors.Wrapf(err, "making test instance %d", i)
		}
		instances[i] = inst
	}
	return &bunchOfSousServers{
		BaseDir:   f.BaseDir,
		Count:     count,
		Instances: instances,
		Stop: func() error {
			return fmt.Errorf("cannot stop bunch of sous servers (not started)")
		},
	}, nil
}

func (c *bunchOfSousServers) configure(t *testing.T, f *fixtureConfig) error {
	siblingURLs := make(map[string]string, c.Count)
	for _, i := range c.Instances {
		siblingURLs[i.ClusterName] = "http://" + i.Addr
	}

	for _, i := range c.Instances {
		if err := i.configure(t, f, siblingURLs); err != nil {
			return errors.Wrapf(err, "configuring instance %d", i)
		}
	}
	return nil
}

func (c *bunchOfSousServers) Start(t *testing.T) {
	t.Helper()
	var started []*sousServer
	// Set the stop func first in case starting returns early.
	c.Stop = func() error {
		var errs []string
		for j, i := range started {
			if err := i.Stop(); err != nil {
				errs = append(errs, fmt.Sprintf(`"could not stop instance%d: %s"`, j, err))
			}
		}
		if len(errs) == 0 {
			return nil
		}
		return fmt.Errorf("could not stop all instances: %s", strings.Join(errs, ", "))
	}
	for _, i := range c.Instances {
		i.Start(t)
		// Note: the value of started is only used in the closure above.
		started = append(started, i)
	}
}
