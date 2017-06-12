package cli

import (
	"flag"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousDeployments is the description of the `sous query gdm` command
type SousDeployments struct {
	GDM               graph.CurrentGDM
	Deployer          sous.Deployer
	Registry          sous.Registry
	State             *sous.State
	DeployFilterFlags config.DeployFilterFlags
	flags             struct {
		singularity string
		registry    string
	}
}

func init() { TopLevelCommands["deployments"] = &SousDeployments{} }

const sousDeployments = `List each deployment of a project managed by sous.

Shows the version that should be deployed according to the manifest,
and if that version does not match reality, shows the actual version
deployed in that clusters as well.`

// Help prints the help
func (*SousDeployments) Help() string { return sousQueryGDMHelp }

// AddFlags adds the flags for sous init.
func (sb *SousDeployments) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sb.DeployFilterFlags, DeployFilterFlagsHelp)
	//MustAddFlags(fs, &sd.OTPLFlags, OtplFlagsHelp)
}

// RegisterOn adds stuff to the graph.
func (sb *SousDeployments) RegisterOn(psy Addable) {
	psy.Add(&sb.DeployFilterFlags)
	psy.Add(graph.DryrunNeither)
}

// Execute defines the behavior of `sous query gdm`
func (sb *SousDeployments) Execute(args []string) cmdr.Result {
	intended := sb.GDM.Deployments
	actual, err := sb.Deployer.RunningDeployments(sb.Registry, sb.State.Defs.Clusters)
	if err != nil {
		return EnsureErrorResult(err)
	}

	intended = intended.Filter(func(d *sous.Deployment) bool {
		return d.SourceID.Location.Repo == sb.DeployFilterFlags.Repo
	})
	actual = actual.Filter(func(d *sous.DeployState) bool {
		return d.SourceID.Location.Repo == sb.DeployFilterFlags.Repo
	})

	type printable struct {
		Cluster, Flavor, Version, Problems string
	}
	var results []printable
	for id, d := range intended.Snapshot() {
		p := printable{
			Cluster: id.Cluster,
			Flavor:  id.ManifestID.Flavor,
			Version: d.SourceID.Version.String(),
		}
		deployState, ok := actual.Get(id)
		if !ok {
			p.Problems = "<not deployed>"
		} else if !deployState.SourceID.Version.Equals(d.SourceID.Version) {
			p.Problems = fmt.Sprintf("actual version deployed: %s", d.SourceID.Version)
		} else {
			p.Problems = "OK"
		}
		if p.Flavor == "" {
			p.Flavor = "<no flavor>"
		}
		results = append(results, p)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "Cluster", "Flavor", "Version", "Status")
	for _, p := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Cluster, p.Flavor, p.Version, p.Problems)
	}

	w.Flush()
	return cmdr.Success()
}
