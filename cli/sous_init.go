package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousInit is the command description for `sous init`
type SousInit struct {
	DeployFilterFlags config.DeployFilterFlags
	Flags             config.OTPLFlags
	Target            graph.TargetManifest
	WD                graph.LocalWorkDirShell
	GDM               graph.CurrentGDM
	State             *sous.State
	StateWriter       graph.StateWriter
	WriteContext      graph.StateWriteContext
}

func init() { TopLevelCommands["init"] = &SousInit{} }

const sousInitHelp = `initialise a new sous project

usage: sous init

Sous init uses contextual information from your current source code tree and
repository to generate a basic configuration for that project. You will need to
flesh out some additional details.

init must be invoked in a git repository that has either an 'upstream' or 
'origin' remote configured.

init will register the project on every known server.`

// Help returns the help string for this command
func (si *SousInit) Help() string { return sousInitHelp }

// RegisterOn adds flag sets for sous init to the dependency injector.
func (si *SousInit) RegisterOn(psy Addable) {
	// Add a zero DepoyFilterFlags to the graph, as we assume a clean build.
	psy.Add(&si.DeployFilterFlags)
	psy.Add(&si.Flags)
}

// AddFlags adds the flags for sous init.
func (si *SousInit) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &si.Flags, OtplFlagsHelp)
	fs.StringVar(&si.DeployFilterFlags.Flavor, "flavor", "", flavorFlagHelp)
	fs.StringVar(&si.DeployFilterFlags.Cluster, "cluster", "", clusterFlagHelp)
}

// Execute fulfills the cmdr.Executor interface
func (si *SousInit) Execute(args []string) cmdr.Result {

	cluster := si.DeployFilterFlags.Cluster

	if _, ok := si.State.Defs.Clusters[cluster]; !ok && cluster != "" {
		return cmdr.UsageErrorf("cluster %q not defined, pick one of: %s", cluster, si.State.Defs.Clusters)
	}

	m := si.Target.Manifest

	if cluster != "" {
		m.Deployments = sous.DeploySpecs{cluster: m.Deployments[cluster]}
	}

	if ok := si.State.Manifests.Add(m); !ok {
		return cmdr.UsageErrorf("manifest %q already exists", m.ID())
	}

	ctx := sous.StateContext(si.WriteContext)
	if err := si.StateWriter.WriteState(si.State, ctx); err != nil {
		return EnsureErrorResult(err)
	}
	return SuccessYAML(m)
}
