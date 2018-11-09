package graph

import (
	"fmt"
	"io"
	"os"

	"github.com/opentable/sous/cli/actions"
	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
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

// GetPlumbingNormalizeGDM returns an update Action.
func (di *SousGraph) GetPlumbingNormalizeGDM() (actions.Action, error) {
	scoop := struct {
		LS     LogSink
		User   sous.User
		Config LocalSousConfig
	}{}
	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}
	return &actions.PlumbNormalizeGDM{
		User:          scoop.User,
		StateLocation: scoop.Config.StateLocation,
		Log:           scoop.LS.LogSink.Child("plumbing-normalize-gdm"),
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

// ArtifactOpts are options for Add Artifacts
type ArtifactOpts struct {
	SourceID    config.SourceIDFlags
	DockerImage string
}

//GetGetArtifact will return artifact for cli add artifact
func (di *SousGraph) GetGetArtifact(opts ArtifactOpts) (actions.Action, error) {
	di.guardedAdd("SourceIDFlags", &opts.SourceID)
	scoop := struct {
		LogSink    LogSink
		User       sous.User
		LocalShell LocalWorkDirShell
		Config     LocalSousConfig
		HTTP       HTTPClient
	}{}
	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}
	return &actions.GetArtifact{

		LogSink:    scoop.LogSink.LogSink.Child("get-artifact"),
		User:       scoop.User,
		Config:     scoop.Config.Config,
		LocalShell: scoop.LocalShell,
		HTTPClient: scoop.HTTP,
		Repo:       opts.SourceID.Repo,
		Offset:     opts.SourceID.Offset,
		Tag:        opts.SourceID.Tag, //might need to switch to version and seperate concept of tag and semv
	}, nil
}

//GetAddArtifact will return artifact for cli add artifact
func (di *SousGraph) GetAddArtifact(opts ArtifactOpts) (actions.Action, error) {
	di.guardedAdd("SourceIDFlags", &opts.SourceID)
	scoop := struct {
		Inserter   sous.ClientInserter
		LogSink    LogSink
		User       sous.User
		LocalShell LocalWorkDirShell
		Config     LocalSousConfig
	}{}
	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}
	return &actions.AddArtifact{
		LogSink:     scoop.LogSink.LogSink.Child("add-artifact"),
		User:        scoop.User,
		Config:      scoop.Config.Config,
		LocalShell:  scoop.LocalShell,
		Inserter:    scoop.Inserter,
		Repo:        opts.SourceID.Repo,
		Offset:      opts.SourceID.Offset,
		Tag:         opts.SourceID.Tag,
		DockerImage: opts.DockerImage,
	}, nil
}

// GetJenkins constructs a Jenkins Actions.
func (di *SousGraph) GetJenkins(opts DeployActionOpts) (actions.Action, error) {
	di.guardedAdd("Dryrun", DryrunOption(opts.DryRun))
	di.guardedAdd("DeployFilterFlags", &opts.DFF)
	scoop := struct {
		HTTP             *ClusterSpecificHTTPClient
		TargetManifestID TargetManifestID
		LogSink          LogSink
		User             sous.User
		Config           LocalSousConfig
		TraceID          sous.TraceID
	}{}
	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}

	client := scoop.HTTP.HTTPClient
	if os.Getenv("SOUS_USE_SOUS_SERVER") == "YES" {
		messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("TraceID: %s", scoop.TraceID), logging.DebugLevel, scoop.LogSink.LogSink, scoop.TraceID)
		c, err := restful.NewClient(scoop.Config.Config.Server, scoop.LogSink.LogSink.Child("jenkins.http-client"), map[string]string{"OT-RequestId": string(scoop.TraceID)})
		if err != nil {
			return nil, err
		}
		client = c
	}

	return &actions.Jenkins{
		HTTPClient:       client,
		TargetManifestID: sous.ManifestID(scoop.TargetManifestID),
		LogSink:          scoop.LogSink.LogSink.Child("jenkins"),
		User:             scoop.User,
		Config:           scoop.Config.Config,
		Cluster:          opts.DFF.Cluster,
	}, nil
}

// BuildActionOpts are options for GetBuild.
type BuildActionOpts struct {
	DFF     config.DeployFilterFlags
	CLIArgs []string
}

// GetBuild gets the Build action.
func (di *SousGraph) GetBuild(opts BuildActionOpts) (*actions.Build, error) {
	scoop := struct {
		ResolveFilter *RefinedResolveFilter
	}{}
	di.MustInject(&scoop)
	opts.DFF.Repo = scoop.ResolveFilter.Repo.ValueOr("")
	getArtifactOpts := ArtifactOpts{
		SourceID: opts.DFF.SourceIDFlags(),
	}
	getArtifact, err := di.GetGetArtifact(getArtifactOpts)
	if err != nil {
		return nil, cmdr.InternalErrorf("%s", err)
	}
	di.guardedAdd("GetArtifact", getArtifact)
	di.guardedAdd("CLIArgs", opts.CLIArgs)
	b := &actions.Build{}
	return b, di.Inject(b)

}

// DeployActionOpts are options for GetDeploy.
type DeployActionOpts struct {
	DFF                              config.DeployFilterFlags
	DryRun, InitSingularityRequestID string
	Force, WaitStable                bool
}

// GetDeploy constructs a Deploy Action.
func (di *SousGraph) GetDeploy(opts DeployActionOpts) (actions.Action, error) {
	di.guardedAdd("Dryrun", DryrunOption(opts.DryRun))
	di.guardedAdd("DeployFilterFlags", &opts.DFF)

	scoop := struct {
		ResolveFilter    *RefinedResolveFilter
		HTTP             *ClusterSpecificHTTPClient
		DeploymentID     TargetDeploymentID
		HTTPStateManager *sous.HTTPStateManager
		LogSink          LogSink
		User             sous.User
		Config           LocalSousConfig
		TraceID          sous.TraceID
	}{}
	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}
	rf := (*sous.ResolveFilter)(scoop.ResolveFilter)

	client := scoop.HTTP.HTTPClient
	if os.Getenv("SOUS_USE_SOUS_SERVER") == "YES" {
		messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("TraceID: %s", scoop.TraceID), logging.DebugLevel, scoop.LogSink.LogSink, scoop.TraceID)
		c, err := restful.NewClient(scoop.Config.Config.Server, scoop.LogSink.LogSink.Child(opts.DFF.Cluster+".http-client"), map[string]string{"OT-RequestId": string(scoop.TraceID)})
		if err != nil {
			return nil, err
		}
		client = c
	}

	did := sous.DeploymentID(scoop.DeploymentID)
	return &actions.Deploy{
		ResolveFilter:      rf,
		HTTPClient:         client,
		TargetDeploymentID: did,
		StateReader:        scoop.HTTPStateManager,
		LogSink:            scoop.LogSink.LogSink.Child("deploy", rf, did),
		User:               scoop.User,
		Config:             scoop.Config.Config,
		Force:              opts.Force,
		WaitStable:         opts.WaitStable,
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
	}{}

	if err := di.Inject(&scoop); err != nil {
		return nil, err
	}

	arScoop := struct {
		AutoResolver *sous.AutoResolver
	}{}

	if enableAutoResolver {
		if err := di.Inject(&arScoop); err != nil {
			return nil, err
		}
	}

	return &actions.Server{
		DeployFilterFlags: dff, // XXX Should be resolve filter
		GDMRepo:           gdmRepo,
		ListenAddr:        laddr,
		Version:           scoop.Version,
		Log:               scoop.LogSink.LogSink,
		Config:            scoop.Config,
		ServerHandler:     scoop.ServerHandler.Handler,
		AutoResolver:      arScoop.AutoResolver,
	}, nil
}
