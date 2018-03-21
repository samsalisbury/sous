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
func (di *SousGraph) GetManifestGet(dff config.DeployFilterFlags, out io.Writer, upCap func(restful.Updater)) (actions.Action, error) {
	di.guardedAdd("DeployFilterFlags", &dff)
	di.guardedAdd("Dryrun", DryrunNeither)

	scoop := struct {
		Dff  config.DeployFilterFlags
		RF   *RefinedResolveFilter
		Tmid TargetManifestID
		HC   HTTPClient
		L    LogSink
	}{}
	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}
	return &actions.ManifestGet{
		DeployFilterFlags: scoop.Dff,
		ResolveFilter:     (*sous.ResolveFilter)(scoop.RF),
		TargetManifestID:  sous.ManifestID(scoop.Tmid),
		HTTPClient:        scoop.HC.HTTPClient,
		LogSink:           scoop.L.LogSink,
		OutWriter:         out,
		UpdaterCapture:    upCap,
	}, nil
}

// GetManifestSet injects a ManifestSet instance.
func (di *SousGraph) GetManifestSet(dff config.DeployFilterFlags, up restful.Updater, in io.Reader) (actions.Action, error) {
	di.guardedAdd("DeployFilterFlags", &dff)
	di.guardedAdd("Dryrun", DryrunNeither)
	scoop := struct {
		Dff  config.DeployFilterFlags
		Tmid TargetManifestID
		HC   HTTPClient
		RF   RefinedResolveFilter
		LS   LogSink
		U    sous.User
	}{}
	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}
	return &actions.ManifestSet{
		DeployFilterFlags: scoop.Dff,
		User:              scoop.U,
		ManifestID:        sous.ManifestID(scoop.Tmid),
		HTTPClient:        scoop.HC.HTTPClient,
		InReader:          in,
		ResolveFilter:     sous.ResolveFilter(scoop.RF),
		LogSink:           scoop.LS.LogSink,
		Updater:           up,
	}, nil
}

// GetUpdate returns an update Action.
func (di *SousGraph) GetUpdate(dff config.DeployFilterFlags, otpl config.OTPLFlags) (actions.Action, error) {
	di.guardedAdd("DeployFilterFlags", &dff)
	di.guardedAdd("OTPLFlags", &otpl)
	di.guardedAdd("Dryrun", DryrunNeither)

	updateScoop := struct {
		Manifest      TargetManifest
		GDM           CurrentGDM
		Client        HTTPClient
		ResolveFilter *RefinedResolveFilter
		User          sous.User
		LogSink       LogSink
	}{}
	if err := di.Inject(&updateScoop); err != nil {
		return nil, err
	}
	return &actions.Update{
		Manifest:      updateScoop.Manifest.Manifest,
		GDM:           updateScoop.GDM.Deployments,
		Client:        updateScoop.Client.HTTPClient,
		ResolveFilter: (*sous.ResolveFilter)(updateScoop.ResolveFilter),
		User:          updateScoop.User,
		Log:           updateScoop.LogSink.LogSink,
	}, nil
}

// GetRectify produces a rectify Action.
func (di *SousGraph) GetRectify(dryrun string, dff config.DeployFilterFlags) (actions.Action, error) {
	di.guardedAdd("Dryrun", DryrunOption(dryrun))
	di.guardedAdd("DeployFilterFlags", &dff)

	scoop := struct {
		LogSink  LogSink
		Resolver *sous.Resolver
		State    *sous.State
	}{}

	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}
	return &actions.Rectify{
		Log:      scoop.LogSink.LogSink,
		Resolver: scoop.Resolver,
		State:    scoop.State,
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
		DeployFilterFlags: dff,
		GDMRepo:           gdmRepo,
		ListenAddr:        laddr,
		Version:           scoop.Version,
		Log:               scoop.LogSink.LogSink,
		Config:            scoop.Config,
		ServerHandler:     scoop.ServerHandler.Handler,
		AutoResolver:      ar,
	}, nil
}
