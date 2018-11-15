package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
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
		noimage bool
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
	fs.BoolVar(&sb.flags.noimage, "noimage", false, "list only deployments that have no registered image")
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

// Execute defines the behavior of `sous query gdm`
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
	if !sb.flags.noimage {
		return sb.dump(deployments)
	}

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

	filteredCount := filtered.Len()
	log.Printf("%d/%d deployments matched your filter", filteredCount, totalCount)
	return sb.dump(filtered)

}
