package cli

import (
	"flag"
	"fmt"
	"os"
	"sort"
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
	TargetManifestID  graph.TargetManifestID
	flags             struct {
		singularity string
		registry    string
	}
}

func init() { TopLevelCommands["deployments"] = &SousDeployments{} }

const sousDeployments = `List each deployment of a project managed by sous.

Shows the version that should be deployed according to the manifest,
and any problems with that deployment.`

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

func (sb *SousDeployments) targetManifest(d *sous.Deployment) bool {
	targetManifestID := sous.ManifestID(sb.TargetManifestID)
	// Filter ignoring flavor if flavor is blank.
	if sb.TargetManifestID.Flavor == "" {
		dummyManifestID := d.ManifestID()
		dummyManifestID.Flavor = ""
		return dummyManifestID == targetManifestID
	}
	return d.ManifestID() == targetManifestID
}

// Execute defines the behavior of `sous query gdm`
func (sb *SousDeployments) Execute(args []string) cmdr.Result {

	fmt.Fprintf(os.Stdout, "Fetching deployments of %s\n\n",
		// Conversion to sous.ManifestID means we get that type's String method.
		sous.ManifestID(sb.TargetManifestID))

	intended := sb.GDM.Deployments
	actualDeployStates, err := sb.Deployer.RunningDeployments(sb.Registry, sb.State.Defs.Clusters)
	if err != nil {
		return EnsureErrorResult(err)
	}

	intended = intended.Filter(sb.targetManifest)
	actual := actualDeployStates.IgnoringStatus().Filter(sb.targetManifest)

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
			p.Problems = fmt.Sprintf("actual version deployed: %s",
				deployState.SourceID.Version)
		} else {
			p.Problems = "OK"
		}
		results = append(results, p)
	}
	// Sort the results by flavor, cluster.
	sort.Slice(results, func(i, j int) bool {
		fi, fj := results[i].Flavor, results[j].Flavor
		ci, cj := results[i].Cluster, results[j].Cluster
		return (fi < fj) || (!(fi > fj) && ci < cj)
	})
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", "Cluster", "Flavor", "Version", "Status")
	for _, p := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Cluster, p.Flavor, p.Version, p.Problems)
	}

	w.Flush()
	return cmdr.Success()
}
