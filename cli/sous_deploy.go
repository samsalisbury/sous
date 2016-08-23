package cli

import (
	"flag"
	"fmt"
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
	StateReader LocalStateReader
	Flags       struct {
		UseOTPLDeploy, IgnoreOTPLDeploy bool
	}
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
	fs.BoolVar(&su.Flags.UseOTPLDeploy, "use-otpl-deploy", false,
		"if specified, copies OpenTable-specific otpl-deploy configuration to the manifest")
	fs.BoolVar(&su.Flags.IgnoreOTPLDeploy, "ignore-otpl-deploy", false,
		"if specified, ignores OpenTable-specific otpl-deploy configuration")
}

// RegisterOn adds the DeploymentConfig to the psyringe to configure the
// labeller and registrar
func (su *SousDeploy) RegisterOn(psy Addable) {
	psy.Add(&su.DeployFilterFlags)
}

// Execute fulfills the cmdr.Executor interface.
func (su *SousDeploy) Execute(args []string) cmdr.Result {
	sid, did, err := getIDs(su.DeployFilterFlags, su.SourceContext.SourceLocation())
	if err != nil {
		return EnsureErrorResult(err)
	}
	_, ok := su.State.Manifests.Get(sid.Location())
	if !ok {
		log.Printf("no manifest for %q; running sous init\n", sid.Location())
		cmdArgs := []interface{}{"init"}
		if su.Flags.UseOTPLDeploy {
			cmdArgs = append(cmdArgs, "-use-otpl-deploy")
		}
		if su.Flags.IgnoreOTPLDeploy {
			cmdArgs = append(cmdArgs, "-ignore-otpl-deploy")
		}
		if err := su.WD.Run("sous", cmdArgs...); err != nil {
			return EnsureErrorResult(err)
		}
		newState, err := su.StateReader.ReadState()
		if err != nil {
			return EnsureErrorResult(err)
		}
		su.State = newState
		newGDM, err := su.State.Deployments()
		if err != nil {
			return EnsureErrorResult(err)
		}
		su.GDM = CurrentGDM{newGDM}
		_, ok := su.State.Manifests.Get(sid.Location())
		if !ok {
			return EnsureErrorResult(fmt.Errorf("sous init failed to add manifest"))
		}
	}
	if err := updateState(su.State, su.GDM, sid, did); err != nil {
		return EnsureErrorResult(err)
	}
	if err := su.StateWriter.WriteState(su.State); err != nil {
		return EnsureErrorResult(err)
	}
	return Success()
}

func updateState(s *sous.State, gdm CurrentGDM, sid sous.SourceID, did sous.DeployID) error {
	deployment, ok := gdm.Get(did)
	if !ok {
		sous.Log.Warn.Printf("Deployment %q does not exist, creating.\n", did)
		deployment = &sous.Deployment{}
	}

	deployment.SourceID = sid
	deployment.ClusterName = did.Cluster

	gdm.Set(did, deployment)

	manifests, err := gdm.Manifests(s.Defs)
	if err != nil {
		return EnsureErrorResult(err)
	}
	s.Manifests = manifests
	return nil
}

func getIDs(flags DeployFilterFlags, sl sous.SourceLocation) (sous.SourceID, sous.DeployID, error) {
	clusterName, tag, sid, did := flags.Cluster, flags.Tag, sous.SourceID{}, sous.DeployID{}
	if clusterName == "" {
		return sid, did, UsageErrorf("You must select a cluster using the -cluster flag.")
	}
	if tag == "" {
		return sid, did, UsageErrorf("You must provide the -tag flag.")
	}
	newVersion, err := semv.Parse(tag)
	if err != nil {
		return sid, did, UsageErrorf("Version %q not valid: %s", flags.Tag, err)
	}
	sid = sl.SourceID(newVersion)
	did = sous.DeployID{Source: sl, Cluster: clusterName}
	return sid, did, nil
}
