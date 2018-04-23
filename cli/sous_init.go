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
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	Flags             config.OTPLFlags         `inject:"optional"`
	// DryRunFlag prints out the manifest but does not save it.
	DryRunFlag  bool `inject:"optional"`
	Target      graph.TargetManifest
	WD          graph.LocalWorkDirShell
	GDM         graph.CurrentGDM
	State       *sous.State
	StateWriter graph.StateWriter
	User        sous.User
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
	psy.Add(graph.DryrunNeither)

	// ugh - there has to be a better way!
	si.Flags.Flavor = si.DeployFilterFlags.Flavor
}

// AddFlags adds the flags for sous init.
func (si *SousInit) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &si.Flags, OtplFlagsHelp)
	fs.StringVar(&si.DeployFilterFlags.Flavor, "flavor", "", flavorFlagHelp)
	fs.StringVar(&si.DeployFilterFlags.Cluster, "cluster", "", clusterFlagHelp)
	fs.StringVar(&si.DeployFilterFlags.Kind, "kind", "", kindFlagHelp)
	fs.BoolVar(&si.DryRunFlag, "dryrun", false, "print out the created manifest but do not save it")
}

// Execute fulfills the cmdr.Executor interface
func (si *SousInit) Execute(args []string) cmdr.Result {

	kind := sous.ManifestKind(si.DeployFilterFlags.Kind)
	kindOk := false

	m := si.Target.Manifest

	switch kind {
	case sous.ManifestKindService:
		kindOk = true
	case sous.ManifestKindScheduled:
		kindOk = true
		for _, v := range m.Deployments {
			v.DeployConfig.Startup.SkipCheck = true
		}
	case sous.ManifestKindOnDemand:
		kindOk = true
	default:
		kindOk = false
	}

	if kindOk == false {
		return cmdr.UsageErrorf("kind not defined, pick one of %s or %s", sous.ManifestKindScheduled, sous.ManifestKindService)
	}

	cluster := si.DeployFilterFlags.Cluster

	if _, ok := si.State.Defs.Clusters[cluster]; !ok && cluster != "" {
		return cmdr.UsageErrorf("cluster %q not defined, pick one of: %s", cluster, si.State.Defs.Clusters)
	}

	m.Kind = kind

	if cluster != "" {
		ds := sous.DeploySpecs{cluster: m.Deployments[cluster]}
		m.Deployments = ds
		//dsc := m.Deployments[cluster]
		//dsc.DeployConfig.Startup = s
	}

	if si.DryRunFlag {
		return SuccessYAML(m)
	}

	if ok := si.State.Manifests.Add(m); !ok {
		return cmdr.UsageErrorf("manifest %q already exists", m.ID())
	}
	if err := si.StateWriter.WriteState(si.State, si.User); err != nil {
		return EnsureErrorResult(err)
	}
	return SuccessYAML(m)
}
