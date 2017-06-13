package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
)

// SousUpdate is the command description for `sous update`
type SousUpdate struct {
	DeployFilterFlags config.DeployFilterFlags
	OTPLFlags         config.OTPLFlags
	Manifest          graph.TargetManifest
	GDM               graph.CurrentGDM
	StateManager      *graph.StateManager
	ResolveFilter     *graph.RefinedResolveFilter
	WriteContext      graph.StateWriteContext
}

func init() { TopLevelCommands["update"] = &SousUpdate{} }

const sousUpdateHelp = `update the version to be deployed in a cluster

usage: sous update -cluster <name> [-tag <semver>] [-use-otpl-deploy|-ignore-otpl-deploy]

sous update will update the version tag for this application in the named
cluster. You can then use 'sous rectify' to have that version deployed.
`

// Help returns the help string for this command
func (su *SousUpdate) Help() string { return sousUpdateHelp }

// AddFlags adds the flags for sous init.
func (su *SousUpdate) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &su.DeployFilterFlags, DeployFilterFlagsHelp)
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
	sid, did, err := getIDs((*sous.ResolveFilter)(su.ResolveFilter), sl)
	if err != nil {
		return EnsureErrorResult(err)
	}

	ctx := sous.StateContext(su.WriteContext)
	gdm, err := updateRetryLoop(su.StateManager.StateManager, sid, did, ctx)
	if err != nil {
		return EnsureErrorResult(err)
	}

	for k, d := range gdm.Snapshot() {
		su.GDM.Set(k, d)
	}

	return cmdr.Success("Updated global manifest.")
}

// If multiple updates are attempted at once for different clusters, there's
// the possibility that they will collide in their updates, either interleaving
// their GDM retreive/manifest update operations, or the git pull/push
// server-side. In this case, the disappointed `sous update` should retry, up
// to the number of times of manifests there are defined for this
// SourceLocation
func updateRetryLoop(sm sous.StateManager, sid sous.SourceID, did sous.DeploymentID, c sous.StateContext) (sous.Deployments, error) {
	tryLimit := 2

	sl := did.ManifestID

	for tries := 0; tries < tryLimit; tries++ {
		state, err := sm.ReadState(c)
		if err != nil {
			return sous.NewDeployments(), err
		}
		manifest, ok := state.Manifests.Get(sl)
		if !ok {
			return sous.NewDeployments(), cmdr.UsageErrorf("No manifest found for %q - try 'sous init' first.", sl)
		}

		tryLimit = len(manifest.Deployments)

		gdm, err := state.Deployments()
		if err != nil {
			return sous.NewDeployments(), err
		}

		if err := updateState(state, gdm, sid, did); err != nil {
			return sous.NewDeployments(), err
		}
		if err := sm.WriteState(state, c); err != nil {
			if !sous.Retryable(err) {
				return sous.NewDeployments(), err
			}

			continue
		}

		return gdm, nil
	}
	return sous.NewDeployments(), errors.Errorf("Tried %d to update %v - %v", tryLimit, sid, did)
}

func updateState(s *sous.State, gdm sous.Deployments, sid sous.SourceID, did sous.DeploymentID) error {
	deployment, ok := gdm.Get(did)
	if !ok {
		sous.Log.Warn.Printf("Deployment %q does not exist, creating.\n", did)
		deployment = &sous.Deployment{}
	}

	deployment.SourceID = sid
	deployment.ClusterName = did.Cluster

	return s.UpdateDeployments(deployment)
}

func getIDs(filter *sous.ResolveFilter, mid sous.ManifestID) (sous.SourceID, sous.DeploymentID, error) {
	var sid sous.SourceID
	var did sous.DeploymentID

	clusterName, tag := filter.Cluster, filter.Tag
	if clusterName == "" {
		return sid, did, cmdr.UsageErrorf("update: You must select a cluster using the -cluster flag.")
	}
	if tag == "" {
		return sid, did, cmdr.UsageErrorf("update: You must provide the -tag flag.")
	}
	newVersion, err := semv.Parse(tag)
	if err != nil {
		return sid, did, cmdr.UsageErrorf("update: Version %q not valid: %s", tag, err)
	}
	sid = mid.Source.SourceID(newVersion)
	did = sous.DeploymentID{ManifestID: mid, Cluster: clusterName}
	return sid, did, nil
}
