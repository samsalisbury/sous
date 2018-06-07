package cli

import (
	"flag"
	"fmt"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
)

// SousInit is the command description for `sous init`
type SousInit struct {
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	Flags             config.OTPLFlags         `inject:"optional"`
	// DryRunFlag prints out the manifest but does not save it.
	DryRunFlag   bool `inject:"optional"`
	Target       graph.TargetManifest
	WD           graph.LocalWorkDirShell
	StateManager *graph.ClientStateManager
	User         sous.User
	flags        struct {
		Kind string
	}

	graph.LogSink
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
	fs.StringVar(&si.flags.Kind, "kind", "", kindFlagHelp)
	fs.BoolVar(&si.DryRunFlag, "dryrun", false, "print out the created manifest but do not save it")
}

// Execute fulfills the cmdr.Executor interface
func (si *SousInit) Execute(args []string) cmdr.Result {

	kind := sous.ManifestKind(si.flags.Kind)
	var skipHealth bool

	switch kind {
	default:
		return cmdr.UsageErrorf("kind %q not defined, pick one of %q, %q or %q", kind, sous.ManifestKindScheduled, sous.ManifestKindService, sous.ManifestKindOnDemand)
	case sous.ManifestKindService:
		skipHealth = false
	case sous.ManifestKindScheduled, sous.ManifestKindOnDemand:
		skipHealth = true
	}

	m := si.Target.Manifest
	if skipHealth {
		for k, d := range m.Deployments {
			// Set the entire 'Startup' so it only has one non-zero field.
			d.Startup = sous.Startup{
				SkipCheck: true,
			}
			m.Deployments[k] = d
		}
	}

	cluster := si.DeployFilterFlags.Cluster

	state, err := si.StateManager.ReadState()
	if err != nil {
		return cmdr.InternalErrorf("getting current state: %s", err)
	}

	logging.Deliver(si.LogSink, logging.ExtraDebug1Level, logging.SousGenericV1, logging.GetCallerInfo(),
		logging.MessageField(fmt.Sprintf("Existing base state: %#v", state)))

	if _, ok := state.Defs.Clusters[cluster]; !ok && cluster != "" {
		return cmdr.UsageErrorf("cluster %q not defined, pick one of: %s", cluster, state.Defs.Clusters)
	}

	m.Kind = kind

	if cluster != "" {
		ds := sous.DeploySpecs{cluster: m.Deployments[cluster]}
		m.Deployments = ds
	}

	if si.DryRunFlag {
		return SuccessYAML(m)
	}

	if ok := state.Manifests.Add(m); !ok {
		return cmdr.UsageErrorf("manifest %q already exists", m.ID())
	}
	logging.Deliver(si.LogSink, logging.ExtraDebug1Level, logging.SousGenericV1, logging.GetCallerInfo(),
		logging.MessageField(fmt.Sprintf("Updated state: %#v", state)))
	if err := si.StateManager.WriteState(state, si.User); err != nil {
		return EnsureErrorResult(err)
	}
	return SuccessYAML(m)
}
