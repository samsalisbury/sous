package cli

import (
	"flag"
	"log"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/samsalisbury/semv"
)

// SousDeploy is the command description for `sous deploy`
type SousDeploy struct {
	DeployFilterFlags DeployFilterFlags
	*sous.SourceContext
	WD          LocalWorkDirShell
	GDM         CurrentGDM
	State       *sous.State
	StateWriter LocalStateWriter
}

func init() { TopLevelCommands["deploy"] = &SousDeploy{} }

const sousUpdateHelp = `
deploy a new version

usage: sous deploy -cluster <name> -tag <semver>
`

// Help returns the help string for this command
func (su *SousDeploy) Help() string { return sousInitHelp }

// AddFlags adds the flags for sous init.
func (su *SousDeploy) AddFlags(fs *flag.FlagSet) {
	err := AddFlags(fs, &su.DeployFilterFlags, rectifyFilterFlagsHelp+tagFlagHelp)
	if err != nil {
		panic(err)
	}
}

// RegisterOn adds the DeploymentConfig to the psyringe to configure the
// labeller and registrar
func (su *SousDeploy) RegisterOn(psy Addable) {
	psy.Add(&su.DeployFilterFlags)
}

// Execute fulfills the cmdr.Executor interface.
func (su *SousDeploy) Execute(args []string) cmdr.Result {
	clusterName, tag := su.DeployFilterFlags.Cluster, su.DeployFilterFlags.Tag
	if clusterName == "" {
		return UsageErrorf("You must a select a cluster using the -cluster flag.")
	}
	if tag == "" {
		return UsageErrorf("you must provide the -tag flag")
	}
	newVersion, err := semv.Parse(tag)
	if err != nil {
		return UsageErrorf("version %q not valid: %s", su.DeployFilterFlags.Tag, err)
	}
	log.Println("USING TAG:", tag)

	sl := su.SourceContext.SourceLocation()
	log.Println("USING SOURCE LOCATION:", sl)

	sid := sl.SourceID(newVersion)
	log.Println("USING SOURCE ID:", sid)

	id := sous.DeployID{Source: sl, Cluster: clusterName}
	log.Println("USING DEPLOY ID:", id)

	deployment, ok := su.GDM.Get(id)
	if !ok {
		log.Printf("Deployment %q does not exist, creating.\n", id)
		for _, k := range su.GDM.Keys() {
			log.Println("EXISTS:", k)
		}
		deployment = &sous.Deployment{}
	}

	deployment.SourceID = id.Source.SourceID(newVersion)
	deployment.ClusterName = clusterName

	su.GDM.Set(id, deployment)

	manifests, err := su.GDM.Manifests(su.State.Defs)
	if err != nil {
		return EnsureErrorResult(err)
	}
	su.State.Manifests = manifests

	if err := su.StateWriter.WriteState(su.State); err != nil {
		return EnsureErrorResult(err)
	}
	return Success()
}
