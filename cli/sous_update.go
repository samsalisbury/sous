package cli

import (
	"flag"
	"fmt"
	"log"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/samsalisbury/semv"
)

// SousUpdate is the command description for `sous update`
type SousUpdate struct {
	config.DeployFilterFlags
	config.OTPLFlags
	Manifest    graph.TargetManifest
	WD          graph.LocalWorkDirShell
	GDM         graph.CurrentGDM
	State       *sous.State
	StateWriter graph.LocalStateWriter
	StateReader graph.LocalStateReader
}

func init() { TopLevelCommands["update"] = &SousUpdate{} }

const sousUpdateHelp = `
update the version to be deployed in a cluster

usage: sous update -cluster <name> -tag <semver> [-use-otpl-deploy|-ignore-otpl-deploy]

sous update will update the version tag for this application in the named
cluster. You can then use 'sous rectify' to have that version deployed.
`

// Help returns the help string for this command
func (su *SousUpdate) Help() string { return sousUpdateHelp }

// AddFlags adds the flags for sous init.
func (su *SousUpdate) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &su.DeployFilterFlags, DeployFilterFlagsHelp)
	MustAddFlags(fs, &su.OTPLFlags, OtplFlagsHelp)
}

// RegisterOn adds the DeploymentConfig to the psyringe to configure the
// labeller and registrar
func (su *SousUpdate) RegisterOn(psy Addable) {
	psy.Add(&su.DeployFilterFlags)
	psy.Add(&su.OTPLFlags)
}

// Execute fulfills the cmdr.Executor interface.
func (su *SousUpdate) Execute(args []string) cmdr.Result {
	sl := su.Manifest.ID()
	sid, did, err := getIDs(su.DeployFilterFlags, sl)
	if err != nil {
		return EnsureErrorResult(err)
	}

	_, ok := su.State.Manifests.Get(sl)
	if !ok {
		log.Printf("adding new  manifest %q", did)
		su.State.Manifests.Add(su.Manifest.Manifest)
		if err := su.StateWriter.WriteState(su.State); err != nil {
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
		su.GDM = graph.CurrentGDM{Deployments: newGDM}
		_, ok := su.State.Manifests.Get(sl)
		if !ok {
			return EnsureErrorResult(fmt.Errorf("failed to add manifest"))
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

func updateState(s *sous.State, gdm graph.CurrentGDM, sid sous.SourceID, did sous.DeployID) error {
	deployment, ok := gdm.Get(did)
	if !ok {
		sous.Log.Warn.Printf("Deployment %q does not exist, creating.\n", did)
		deployment = &sous.Deployment{}
	}

	deployment.SourceID = sid
	deployment.ClusterName = did.Cluster

	// XXX switch to .UpdateDeployments
	gdm.Set(did, deployment)

	manifests, err := gdm.Manifests(s.Defs)
	if err != nil {
		return EnsureErrorResult(err)
	}
	s.Manifests = manifests
	return nil
}

func getIDs(flags config.DeployFilterFlags, mid sous.ManifestID) (sous.SourceID, sous.DeployID, error) {
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
	sid = mid.Source.SourceID(newVersion)
	did = sous.DeployID{ManifestID: mid, Cluster: clusterName}
	return sid, did, nil
}
