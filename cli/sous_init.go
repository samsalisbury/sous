package cli

import (
	"flag"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousInit is the command description for `sous init`
type SousInit struct {
	DeployFilterFlags
	Flags         OTPLFlags
	Target        TargetManifest
	SourceContext *sous.SourceContext
	WD            LocalWorkDirShell
	GDM           CurrentGDM
	State         *sous.State
	StateWriter   LocalStateWriter
}

func init() { TopLevelCommands["init"] = &SousInit{} }

const sousInitHelp = `
initialise a new sous project

usage: sous init

Sous init uses contextual information from your current source code tree and
repository to generate a basic configuration for that project. You will need to
flesh out some additional details.
`

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
	MustAddFlags(fs, &si.Flags, otplFlagsHelp)
	fs.StringVar(&si.DeployFilterFlags.Flavor, "flavor", "", flavorFlagHelp)
}

// Execute fulfills the cmdr.Executor interface
func (si *SousInit) Execute(args []string) cmdr.Result {
	m := si.Target.Manifest
	if ok := si.State.Manifests.Add(m); !ok {
		return UsageErrorf("manifest %q already exists", m.ID())
	}
	if err := si.StateWriter.WriteState(si.State); err != nil {
		return EnsureErrorResult(err)
	}
	return SuccessYAML(m)
}
