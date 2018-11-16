package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousQueryGDM is the description of the `sous query gdm` command
type SousQueryGDM struct {
	StateManager *graph.ClientStateManager
	flags        struct {
		filters string
		format  string
	}
	SousGraph *graph.SousGraph
}

func init() { QuerySubcommands["gdm"] = &SousQueryGDM{} }

const sousQueryGDMHelp = `The intended state of deployment for every project and every cluster known to Sous.

The results of 'sous query gdm' and 'sous query ads' will not be identical if
a problem is preventing sous from modifying the current state of Singularity.
`

// Help prints the help
func (*SousQueryGDM) Help() string { return sousQueryGDMHelp }

// RegisterOn adds options set by flags to the injection graph.
func (*SousQueryGDM) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&config.DeployFilterFlags{})
}

// AddFlags adds the flags for 'sous query gdm'.
func (sb *SousQueryGDM) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&sb.flags.filters, "filters", "", "filter the output, space-separatey list, e.g. 'hasimage=true zeroinstances=false")
	fs.StringVar(&sb.flags.format, "format", "table", "output format, one of (table, json)")
}

func (sb *SousQueryGDM) dump(ds sous.Deployments) cmdr.Result {
	var err error
	switch sb.flags.format {
	default:
		err = fmt.Errorf("output format %q not allowed, pick one of: table, json", sb.flags.format)
		fallthrough
	case "table":
		sous.DumpDeployments(os.Stdout, ds)
	case "json":
		sous.JSONDeployments(os.Stdout, ds)
	}
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	return cmdr.Success()
}

type deployFilter func(sous.Deployments, bool) sous.Deployments
type boundFilter func(sous.Deployments) sous.Deployments

func (sb *SousQueryGDM) availableFilters() map[string]deployFilter {
	return map[string]deployFilter{
		"hasimage":      sb.hasImageFilter,
		"zeroinstances": zeroInstanceFilter,
	}
}

func (sb *SousQueryGDM) hasImageFilter(deployments sous.Deployments, which bool) sous.Deployments {
	filtered := sous.NewDeployments()
	wg := sync.WaitGroup{}

	ds := deployments.Snapshot()

	wg.Add(len(ds))
	errs := make(chan error, len(ds))
	getArtifactMutex := sync.Mutex{}

	for _, d := range ds {
		d := d
		go func() {
			defer wg.Done()
			opts := graph.ArtifactOpts{
				SourceID: config.NewSourceIDFlags(d.SourceID),
			}
			getArtifactMutex.Lock()
			getArtifact, err := sb.SousGraph.GetGetArtifact(opts)
			getArtifactMutex.Unlock()
			if err != nil {
				errs <- err
				return
			}
			exists, err := getArtifact.ArtifactExists()
			if err != nil {
				errs <- err
				return
			}
			if exists {
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

func zeroInstanceFilter(ds sous.Deployments, which bool) sous.Deployments {
	return ds.Filter(func(d *sous.Deployment) bool {
		return d.NumInstances == 0 == which
	})
}

func (sb *SousQueryGDM) getFilter(name string) (deployFilter, error) {
	f, ok := sb.availableFilters()[name]
	if !ok {
		return nil, fmt.Errorf("filter %q not recognised; pick either hasimage or zeroinstances", name)
	}
	return f, nil
}

func (sb *SousQueryGDM) parseFilters() ([]boundFilter, error) {
	var filters []boundFilter
	if sb.flags.filters == "" {
		return nil, nil
	}
	parts := strings.Fields(sb.flags.filters)
	for _, p := range parts {
		kv := strings.Split(p, "=")
		if len(kv) != 2 {
			return nil, cmdr.UsageErrorf("filter %q not valid; format is name=(true|false)")
		}
		k, v := kv[0], kv[1]
		f, err := sb.getFilter(k)
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

func (sb *SousQueryGDM) filter(ds sous.Deployments) (sous.Deployments, error) {
	filters, err := sb.parseFilters()
	if err != nil {
		return sous.NewDeployments(), err
	}
	for _, f := range filters {
		ds = f(ds)
	}
	return ds, nil
}

// Execute defines the behavior of `sous query gdm`.
func (sb *SousQueryGDM) Execute(args []string) cmdr.Result {

	state, err := sb.StateManager.ReadState()
	if err != nil {
		return EnsureErrorResult(err)
	}
	deployments, err := state.Deployments()

	totalCount := deployments.Len()

	if err != nil {
		return EnsureErrorResult(err)
	}

	filtered, err := sb.filter(deployments)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	filteredCount := filtered.Len()
	log.Printf("%d results (of %q total deployments)", filteredCount, totalCount)
	return sb.dump(filtered)

}
