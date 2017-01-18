package graph

import (
	"fmt"
	"io"
	"io/ioutil"
	"log" //ok
	"os"
	"os/user"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/git"
	"github.com/opentable/sous/ext/github"
	"github.com/opentable/sous/ext/singularity"
	"github.com/opentable/sous/ext/storage"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/shell"
	"github.com/pkg/errors"
	"github.com/samsalisbury/psyringe"
	"github.com/samsalisbury/semv"
)

type (
	// Out is an output used for real data a Command returns. This should only
	// be used when a command needs to write directly to stdout, using the
	// formatting options that come with an output. Usually, you should use a
	// SuccessResult with Data to return data.
	Out struct{ *cmdr.Output }
	// ErrOut is an output used for logging from a Command. This should only be
	// used when a Command needs to write a lot of data to stderr, using the
	// formatting options that come with and Output. Usually you should use and
	// ErrorResult to return error messages.
	ErrOut struct{ *cmdr.Output }
	// SousGraph is a dependency injector used to flesh out Sous commands
	// with their dependencies.
	SousGraph struct{ *psyringe.Psyringe }
	// OutWriter is typically set to os.Stdout.
	OutWriter io.Writer
	// ErrWriter is typically set to os.Stderr.
	ErrWriter io.Writer
	// InReader is typicially set to os.Stdin
	InReader io.Reader
	// StatusWaitStable represents if `sous plumbing status` should continue to
	// poll until the selected t
	StatusWaitStable bool
	// Version represents a version of Sous.
	Version struct{ semv.Version }
	// LocalUser is the currently logged in user.
	LocalUser struct{ *config.User }
	// LocalSousConfig is the configuration for Sous.
	LocalSousConfig struct{ *config.Config }
	// LocalWorkDir is the user's current working directory when they invoke Sous.
	LocalWorkDir string
	// LocalWorkDirShell is a shell for working in the user's current working
	// directory.
	LocalWorkDirShell struct{ *shell.Sh }
	// LocalGitClient is a git client rooted in WorkdirShell.Dir.
	LocalGitClient struct{ *git.Client }
	// LocalGitRepo is the git repository containing WorkDir.
	LocalGitRepo struct{ *git.Repo }
	// ScratchDirShell is a shell for working in the scratch area where things
	// like artefacts, and build metadata are stored. It is a new, empty
	// directory, and should be cleaned up eventually.
	ScratchDirShell struct{ *shell.Sh }
	// LocalDockerClient is a docker client object
	LocalDockerClient struct{ docker_registry.Client }
	// StateManager simply wraps the sous.StateManager interface
	StateManager struct{ sous.StateManager }
	// StateReader wraps a storage.StateReader.
	StateReader struct{ sous.StateReader }
	// StateWriter wraps a storage.StateWriter, and should be configured to
	// use the current user's local storage.
	StateWriter struct{ sous.StateWriter }
	// CurrentGDM is a snapshot of the GDM at application start. In a CLI
	// context, which this is, that is all we need to simply read the GDM.
	CurrentGDM struct{ sous.Deployments }
	// TargetManifest is a specific manifest for the current ManifestID.
	// If the named manifest does not exist, it is created.
	TargetManifest struct{ *sous.Manifest }
	// detectedOTPLDeployManifest is a set of otpl-deploy configured deployments
	// that have been detected.
	detectedOTPLDeployManifest struct{ sous.DeploySpecs }
	// userSelectedOTPLDeployManifest is a set of otpl-deploy configured deploy
	// specs that the user has explicitly selected. (May be empty.)
	userSelectedOTPLDeployManifest struct{ sous.DeploySpecs }
	// TargetManifestID is the manifest ID being targeted, after resolving all
	// context and flags.
	TargetManifestID sous.ManifestID
	// DryrunOption specifies components that should be faked in an execution.
	DryrunOption string
	// SourceContextDiscovery captures the possiblity of not finding a SourceContext
	SourceContextDiscovery struct {
		Error error
		*sous.ManifestID
		*sous.SourceContext
	}
)

const (
	// DryrunBoth specifies that both the registry and scheduler should be fake.
	DryrunBoth = DryrunOption("both")
	// DryrunRegistry means the registry should be faked for this execution.
	DryrunRegistry = DryrunOption("registry")
	// DryrunScheduler means the scheduler should be faked for this execution.
	DryrunScheduler = DryrunOption("scheduler")
	// DryrunNeither means neither the registry or scheduler should be faked.
	DryrunNeither = DryrunOption("none")
)

// BuildGraph builds the dependency injection graph, used to populate commands
// invoked by the user.
func BuildGraph(in io.Reader, out, err io.Writer) *SousGraph {
	graph := buildBaseGraph(in, out, err)
	AddFilesystem(graph)
	return graph
}

func buildBaseGraph(in io.Reader, out, err io.Writer) *SousGraph {
	graph := &SousGraph{psyringe.New()}
	// stdout, stderr
	graph.Add(
		func() InReader { return in },
		func() OutWriter { return out },
		func() ErrWriter { return err },
	)

	AddLogs(graph)
	AddUser(graph)
	AddShells(graph)
	AddConfig(graph)
	AddNetwork(graph)
	AddDocker(graph)
	AddSingularity(graph)
	AddState(graph)
	AddInternals(graph)
	return graph
}

type adder interface {
	Add(...interface{})
}

// AddLogs adds a logset to the graph.
func AddLogs(graph adder) {
	graph.Add(
		newLogSet,
	)
}

// AddUser adds the OS user to the graph.
func AddUser(graph adder) {
	graph.Add(
		newLocalUser,
	)
}

// AddShells adds working shells to the graph.
func AddShells(graph adder) {
	graph.Add(
		newLocalWorkDirShell,
		newScratchDirShell,
	)
}

// AddFilesystem adds filesystem to the graph.
func AddFilesystem(graph adder) {
	graph.Add(
		newConfigLoader,
	)
}

// AddConfig adds filesystem to the graph.
func AddConfig(graph adder) {
	c := config.DefaultConfig()
	graph.Add(
		newPossiblyInvalidLocalSousConfig,
		DefaultConfig{&c},
		newLocalSousConfig,
		newSousConfig,
		newLocalWorkDir,
	)
}

// AddNetwork adds features that require the network.
func AddNetwork(graph adder) {
	graph.Add(
		newDockerClient,
		newHTTPClient,
		newStatusPoller,
	)
}

// AddDocker adds Docker to the graph.
func AddDocker(graph adder) {
	graph.Add(
		newDockerBuilder,
		newSelector,
	)
}

// AddSingularity adds Singularity clients to the graph.
func AddSingularity(graph adder) {
	graph.Add(
		newDeployer,
	)
}

// AddState adds state reader and writers to the graph.
func AddState(graph adder) {
	graph.Add(
		newStateManager,
		newLocalStateReader,
		newLocalStateWriter,
	)
}

// AddInternals adds the dependency contructors that are internal to Sous.
func AddInternals(graph adder) {
	// internal to Sous
	graph.Add(
		newRegistryDumper,
		newRegistry,
		newLabeller,
		newRegistrar,
		newBuildManager,
		newBuildConfig,
		newBuildContext,
		newSourceContext,
		newSourceContextDiscovery,
		newLocalGitClient,
		newLocalGitRepo,
		newSourceHostChooser,
		NewCurrentState,
		NewCurrentGDM,
		newTargetManifest,
		newDetectedOTPLConfig,
		newUserSelectedOTPLDeploySpecs,
		newTargetManifestID,
		newResolveFilter,
		newResolver,
		newAutoResolver,
		newInserter,
	)
}

func newResolveFilter(sf *config.DeployFilterFlags, shc sous.SourceHostChooser) (*sous.ResolveFilter, error) {
	return sf.BuildFilter(shc.ParseSourceLocation)
}

func newResolver(filter *sous.ResolveFilter, d sous.Deployer, r sous.Registry) *sous.Resolver {
	return sous.NewResolver(d, r, filter)
}

func newAutoResolver(rez *sous.Resolver, sr StateReader, ls *sous.LogSet) *sous.AutoResolver {
	return sous.NewAutoResolver(rez, sr, ls)
}

func newSourceHostChooser() sous.SourceHostChooser {
	return sous.SourceHostChooser{
		SourceHosts: []sous.SourceHost{
			github.SourceHost{},
		},
	}
}

func newRegistryDumper(r sous.Registry) *sous.RegistryDumper {
	return sous.NewRegistryDumper(r)
}

func newLogSet(v *config.Verbosity, err ErrWriter) *sous.LogSet { // XXX temporary until we settle on logging
	sous.Log.Info.SetOutput(err)

	if v.Debug {
		if v.Loud {
			sous.Log.Vomit.SetOutput(err)
		}
		sous.Log.Debug.SetOutput(err)
	}
	if v.Loud {
	}
	if v.Quiet {
		sous.Log.Info.SetOutput(ioutil.Discard)
	}
	if v.Silent {
		sous.Log.Info.SetOutput(ioutil.Discard)
	}

	//sous.Log.Warn.Println("Normal output enabled")
	sous.Log.Vomit.Println("Verbose debugging enabled")
	sous.Log.Debug.Println("Regular debugging enabled")
	return &sous.Log
}

func newSourceContextDiscovery(g LocalGitRepo) *SourceContextDiscovery {
	c, err := g.SourceContext()
	return &SourceContextDiscovery{
		Error:         err,
		SourceContext: c,
	}
}

func newSourceContext(mid TargetManifestID, discovered *SourceContextDiscovery) (*sous.SourceContext, error) {
	return discovered.Unwrap(mid)
}

// Unwrap returns the SourceContext and the returned error in trying to create it.
func (scd *SourceContextDiscovery) Unwrap(mid TargetManifestID) (*sous.SourceContext, error) {
	if scd.Error != nil {
		return nil, scd.Error
	}
	sl := sous.ManifestID(mid)
	if sl.Source.Repo != scd.SourceLocation().Repo {
		return nil, errors.Errorf("source location %q is not the same as the remote %q", sl.Source.Repo, scd.SourceLocation().Repo)
	}
	return scd.SourceContext, nil
}

// GetContext returns the SourceContext discovered if there were no errors in
// getting it. Otherwise returns a pointer to a zero SourceContext.
func (scd *SourceContextDiscovery) GetContext() *sous.SourceContext {
	if scd.Error != nil || scd.SourceContext == nil {
		return &sous.SourceContext{}
	}
	return scd.SourceContext
}

func newBuildContext(wd LocalWorkDirShell, c *sous.SourceContext) *sous.BuildContext {
	return &sous.BuildContext{Sh: wd.Sh, Source: *c}
}

func newBuildConfig(f *config.DeployFilterFlags, p *config.PolicyFlags, bc *sous.BuildContext) *sous.BuildConfig {
	offset := f.Offset
	if offset == "" {
		offset = bc.Source.OffsetDir
	}
	cfg := sous.BuildConfig{
		Repo:       f.Repo,
		Offset:     offset,
		Tag:        f.Tag,
		Revision:   f.Revision,
		Strict:     p.Strict,
		ForceClone: p.ForceClone,
		Context:    bc,
	}
	cfg.Resolve()

	return &cfg
}

func newBuildManager(bc *sous.BuildConfig, sl sous.Selector, lb sous.Labeller, rg sous.Registrar) *sous.BuildManager {
	return &sous.BuildManager{
		BuildConfig: bc,
		Selector:    sl,
		Labeller:    lb,
		Registrar:   rg,
	}
}

func newLocalUser() (v LocalUser, err error) {
	u, err := user.Current()
	v.User = &config.User{User: u}
	return v, initErr(err, "getting current user")
}

// TODO: This should register a cleanup task with the cli, to delete the temp
// dir.
func newScratchDirShell() (v ScratchDirShell, err error) {
	const what = "getting scratch directory"
	dir, err := ioutil.TempDir("", "sous")
	if err != nil {
		return v, initErr(err, what)
	}
	v.Sh, err = shell.DefaultInDir(dir)
	v.TeeOut = os.Stdout
	v.TeeErr = os.Stderr
	return v, initErr(err, what)
}

func newLocalWorkDir() (LocalWorkDir, error) {
	s, err := os.Getwd()
	return LocalWorkDir(s), initErr(err, "determining working directory")
}

func newLocalWorkDirShell(verbosity *config.Verbosity, l LocalWorkDir) (v LocalWorkDirShell, err error) {
	v.Sh, err = shell.DefaultInDir(string(l))
	v.TeeEcho = os.Stdout //XXX should use a writer
	v.Sh.Debug = verbosity.Debug
	//v.TeeOut = os.Stdout
	//v.TeeErr = os.Stderr
	return v, initErr(err, "getting current working directory")
}

func newLocalGitClient(sh LocalWorkDirShell) (v LocalGitClient, err error) {
	v.Client, err = git.NewClient(sh.Sh)
	return v, initErr(err, "initialising git client")
}

func newLocalGitRepo(c LocalGitClient) (v LocalGitRepo, err error) {
	v.Repo, err = c.OpenRepo(".")
	return v, initErr(err, "opening local git repository")
}

func newSelector() sous.Selector {
	return &sous.EchoSelector{
		Factory: func(*sous.BuildContext) (sous.Buildpack, error) {
			return docker.NewDockerfileBuildpack(), nil
		},
	}
}

func newDockerBuilder(cfg LocalSousConfig, cl LocalDockerClient, ctx *sous.SourceContext, source LocalWorkDirShell, scratch ScratchDirShell) (*docker.Builder, error) {
	nc, err := makeDockerRegistry(cfg, cl)
	if err != nil {
		return nil, err
	}
	drh := cfg.Docker.RegistryHost
	source.Sh = source.Sh.Clone().(*shell.Sh)
	source.Sh.LongRunning = true
	return docker.NewBuilder(nc, drh, source.Sh, scratch.Sh)
}

func newLabeller(db *docker.Builder) sous.Labeller {
	return db
}

func newRegistrar(db *docker.Builder) sous.Registrar {
	return db
}

func newRegistry(dryrun DryrunOption, cfg LocalSousConfig, cl LocalDockerClient) (sous.Registry, error) {
	if dryrun == DryrunBoth || dryrun == DryrunRegistry {
		return sous.NewDummyRegistry(), nil
	}
	return makeDockerRegistry(cfg, cl)
}

func newDeployer(dryrun DryrunOption) sous.Deployer {
	// Eventually, based on configuration, we may make different decisions here.
	if dryrun == DryrunBoth || dryrun == DryrunScheduler {
		drc := sous.NewDummyRectificationClient()
		drc.SetLogger(log.New(os.Stdout, "rectify: ", 0))
		return singularity.NewDeployer(drc)
	}
	return singularity.NewDeployer(singularity.NewRectiAgent())
}

func newDockerClient() LocalDockerClient {
	return LocalDockerClient{docker_registry.NewClient()}
}

// newHTTPClient returns an HTTP client if c.Server is not empty.
// Otherwise it returns nil, and emits some warnings.
func newHTTPClient(c LocalSousConfig) (*sous.HTTPClient, error) {
	if c.Server == "" {
		sous.Log.Warn.Println("No server set, Sous is running in server or workstation mode.")
		sous.Log.Warn.Println("Configure a server like this: sous config server http://some.sous.server")
		return nil, nil
	}
	sous.Log.Debug.Printf("Using server at %s", c.Server)
	return sous.NewClient(c.Server)
}

// newStateManager returns a wrapped sous.HTTPStateManager if cl is not nil.
// Otherwise it returns a wrapped sous.GitStateManager, for local git based GDM.
// If it returns a sous.GitStateManager, it emits a warning log.
func newStateManager(cl *sous.HTTPClient, c LocalSousConfig) *StateManager {
	if cl == nil {
		sous.Log.Warn.Printf("Using local state stored at %s", c.StateLocation)
		dm := storage.NewDiskStateManager(c.StateLocation)
		return &StateManager{StateManager: storage.NewGitStateManager(dm)}
	}
	hsm := sous.NewHTTPStateManager(cl)
	return &StateManager{StateManager: hsm}
}

func newStatusPoller(cl *sous.HTTPClient, rf *sous.ResolveFilter) *sous.StatusPoller {
	if cl == nil {
		sous.Log.Warn.Println("Unable to poll for status.")
		return nil
	}
	return sous.NewStatusPoller(cl, rf)
}

func newLocalStateReader(sm *StateManager) StateReader {
	return StateReader{sm}
}

func newLocalStateWriter(sm *StateManager) StateWriter {
	return StateWriter{sm}
}

// NewCurrentState returns the current *sous.State.
func NewCurrentState(sr StateReader) (*sous.State, error) {
	state, err := sr.ReadState()
	if os.IsNotExist(errors.Cause(err)) {
		log.Println("error reading state:", err)
		log.Println("defaulting to empty state")
		return sous.NewState(), nil
	}
	return state, initErr(err, "reading sous state")
}

// NewCurrentGDM returns the current GDM.
func NewCurrentGDM(state *sous.State) (CurrentGDM, error) {
	deployments, err := state.Deployments()
	if err != nil {
		return CurrentGDM{}, initErr(err, "expanding state")
	}
	return CurrentGDM{deployments}, initErr(err, "expanding state")
}

// The funcs named makeXXX below are used to create specific implementations of
// sous native types.

// makeDockerRegistry creates a Docker version of sous.Registry
func makeDockerRegistry(cfg LocalSousConfig, cl LocalDockerClient) (*docker.NameCache, error) {
	dbCfg := cfg.Docker.DBConfig()
	db, err := docker.GetDatabase(&dbCfg)
	if err != nil {
		return nil, errors.Wrap(err, "building name cache DB")
	}
	drh := cfg.Docker.RegistryHost
	return docker.NewNameCache(drh, cl.Client, db), nil
}

func newInserter(cfg LocalSousConfig, cl LocalDockerClient) (sous.Inserter, error) {
	if cfg.Server == "" {
		return makeDockerRegistry(cfg, cl)
	}
	return sous.NewHTTPNameInserter(cfg.Server)
}

// initErr returns nil if error is nil, otherwise an initialisation error.
// The second argument "what" should be a very short description of the
// initialisation task, e.g. "getting widget" or "reading state" etc.
func initErr(err error, what string) error {
	if err == nil {
		return nil
	}
	message := fmt.Sprintf("error %s:", what)
	if shellErr, ok := err.(shell.Error); ok {
		message += fmt.Sprintf("\ncommand failed:\nshell> %s\n%s",
			shellErr.Command.String(), shellErr.Result.Combined.String())
	} else {
		message += " " + err.Error()
	}
	return fmt.Errorf(message)
}
