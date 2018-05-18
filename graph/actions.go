package graph

import (
	"io"
	"os"

	"github.com/opentable/sous/cli/actions"
	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
	"github.com/samsalisbury/semv"
)

func (di *SousGraph) guardedAdd(guardName string, value interface{}) {
	if di.addGuards[guardName] {
		return
	}
	di.addGuards[guardName] = true
	di.Add(value)
}

// GetManifestGet injects a ManifestGet instances.
func (di *SousGraph) GetManifestGet(dff config.DeployFilterFlags, out io.Writer, upCap *restful.Updater) (actions.Action, error) {
	di.guardedAdd("DeployFilterFlags", &dff)
	di.guardedAdd("Dryrun", DryrunNeither)

	scoop := struct {
		RF   *RefinedResolveFilter
		Tmid TargetManifestID
		HC   HTTPClient
		L    LogSink
	}{}
	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}

	rf := (*sous.ResolveFilter)(scoop.RF)
	mid := sous.ManifestID(scoop.Tmid)
	return &actions.ManifestGet{
		ResolveFilter:    rf,
		TargetManifestID: mid,
		HTTPClient:       scoop.HC.HTTPClient,
		LogSink:          scoop.L.LogSink.Child("manifest-get", rf, mid),
		OutWriter:        out,
		UpdaterCapture:   upCap,
	}, nil
}

// GetManifestSet injects a ManifestSet instance.
func (di *SousGraph) GetManifestSet(dff config.DeployFilterFlags, up *restful.Updater, in io.Reader) (actions.Action, error) {
	di.guardedAdd("DeployFilterFlags", &dff)
	di.guardedAdd("Dryrun", DryrunNeither)
	scoop := struct {
		Tmid TargetManifestID
		RF   *RefinedResolveFilter
		LS   LogSink
		U    sous.User
	}{}
	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}
	mid := sous.ManifestID(scoop.Tmid)
	rf := (*sous.ResolveFilter)(scoop.RF)
	return &actions.ManifestSet{
		User:          scoop.U,
		ManifestID:    mid,
		InReader:      in,
		ResolveFilter: rf,
		LogSink:       scoop.LS.LogSink.Child("manifest-set", rf, mid),
		Updater:       up,
	}, nil
}

// GetUpdate returns an update Action.
func (di *SousGraph) GetUpdate(dff config.DeployFilterFlags, otpl config.OTPLFlags) (actions.Action, error) {
	di.guardedAdd("DeployFilterFlags", &dff)
	di.guardedAdd("OTPLFlags", &otpl)
	di.guardedAdd("Dryrun", DryrunNeither)

	scoop := struct {
		Manifest         TargetManifest
		GDM              CurrentGDM
		HTTPStateManager *sous.HTTPStateManager
		ResolveFilter    *RefinedResolveFilter
		User             sous.User
		LogSink          LogSink
	}{}
	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}
	return &actions.Update{
		Manifest:         scoop.Manifest.Manifest,
		GDM:              scoop.GDM.Deployments,
		HTTPStateManager: scoop.HTTPStateManager,
		ResolveFilter:    (*sous.ResolveFilter)(scoop.ResolveFilter),
		User:             scoop.User,
		Log:              scoop.LogSink.LogSink,
	}, nil
}

// GetDeploy constructs a Deploy Actions.
func (di *SousGraph) GetDeploy(dff config.DeployFilterFlags, dryrun string, force, waitStable bool) (actions.Action, error) {
	di.guardedAdd("Dryrun", DryrunOption(dryrun))
	di.guardedAdd("DeployFilterFlags", &dff)

	scoop := struct {
		ResolveFilter    *RefinedResolveFilter
		HTTP             *ClusterSpecificHTTPClient
		DeploymentID     TargetDeploymentID
		HTTPStateManager *sous.HTTPStateManager
		LogSink          LogSink
		User             sous.User
		Config           LocalSousConfig
	}{}
	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}
	rf := (*sous.ResolveFilter)(scoop.ResolveFilter)
	did := sous.DeploymentID(scoop.DeploymentID)
	return &actions.Deploy{
		ResolveFilter:      rf,
		HTTPClient:         scoop.HTTP.HTTPClient,
		TargetDeploymentID: did,
		StateReader:        scoop.HTTPStateManager,
		LogSink:            scoop.LogSink.LogSink.Child("deploy", rf, did),
		User:               scoop.User,
		Config:             scoop.Config.Config,
		Force:              force,
		WaitStable:         waitStable,
	}, nil
}

// GetRectify produces a rectify Action.
func (di *SousGraph) GetRectify(dryrun string, dff config.DeployFilterFlags) (actions.Action, error) {
	di.guardedAdd("Dryrun", DryrunOption(dryrun))
	di.guardedAdd("DeployFilterFlags", &dff)

	scoop := struct {
		LogSink  LogSink
		Resolver *sous.Resolver
		SM       *ServerStateManager
	}{}

	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}

	state, err := scoop.SM.ReadState()
	if err != nil {
		return nil, err
	}

	return &actions.Rectify{
		Log:      scoop.LogSink.LogSink,
		Resolver: scoop.Resolver,
		State:    state,
	}, nil
}

// GetPollStatus produces an Action to poll the status of a deployment.
func (di *SousGraph) GetPollStatus(dryrun string, dff config.DeployFilterFlags) (actions.Action, error) {
	di.guardedAdd("Dryrun", DryrunOption(dryrun))
	di.guardedAdd("DeployFilterFlags", &dff)

	scoop := struct {
		StatusPoller *sous.StatusPoller
	}{}

	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}

	return &actions.PollStatus{
		StatusPoller: scoop.StatusPoller,
	}, nil
}

// GetServer returns the server action.
func (di *SousGraph) GetServer(
	dff config.DeployFilterFlags,
	dryrun string,
	laddr string,
	gdmRepo string,
	profiling bool,
	enableAutoResolver bool,
) (actions.Action, error) {
	dff.Offset = "*"
	dff.Flavor = "*"
	profiling = profiling || os.Getenv("SOUS_PROFILING") == "enable"

	di.guardedAdd("DeployFilterFlags", &dff)
	di.guardedAdd("Dryrun", DryrunOption(dryrun))
	di.guardedAdd("ProfilingServer", ProfilingServer(profiling))

	scoop := struct {
		Version       semv.Version
		LogSink       LogSink
		Config        *config.Config
		ServerHandler ServerHandler
		AutoResolver  *sous.AutoResolver
	}{}

	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}

	ar := scoop.AutoResolver
	if !enableAutoResolver {
		ar = nil
	}

	return &actions.Server{
		DeployFilterFlags: dff, // XXX Should be resolve filter
		GDMRepo:           gdmRepo,
		ListenAddr:        laddr,
		Version:           scoop.Version,
		Log:               scoop.LogSink.LogSink,
		Config:            scoop.Config,
		ServerHandler:     scoop.ServerHandler.Handler,
		AutoResolver:      ar,
	}, nil
}
