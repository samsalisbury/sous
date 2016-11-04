package graph

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
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
	// GitSourceContext is the source context according to the local git repo.
	GitSourceContext struct{ *sous.SourceContext }
	// ScratchDirShell is a shell for working in the scratch area where things
	// like artefacts, and build metadata are stored. It is a new, empty
	// directory, and should be cleaned up eventually.
	ScratchDirShell struct{ *shell.Sh }
	// LocalDockerClient is a docker client object
	LocalDockerClient struct{ docker_registry.Client }
	// StateManager simply wraps the sous.StateManager interface
	StateManager struct{ sous.StateManager }
	// LocalStateReader wraps a storage.StateReader, and should be configured
	// to use the current user's local storage.
	LocalStateReader struct{ sous.StateReader }
	// LocalStateWriter wraps a storage.StateWriter, and should be configured to
	// use the current user's local storage.
	LocalStateWriter struct{ sous.StateWriter }
	// CurrentGDM is a snapshot of the GDM at application start. In a CLI
	// context, which this is, that is all we need to simply read the GDM.
	CurrentGDM struct{ sous.Deployments }
	// TargetManifest is a specific manifest for the current ManifestID.
	// If the named manifest does not exist, it is created.
	TargetManifest struct{ *sous.Manifest }
	// DetectedOTPLDeploySpecs is a set of otpl-deploy configured deployments
	// that have been detected.
	DetectedOTPLDeploySpecs struct{ sous.DeploySpecs }
	// UserSelectedOTPLDeploySpecs is a set of otpl-deploy configured deploy
	// specs that the user has explicitly selected. (May be empty.)
	UserSelectedOTPLDeploySpecs struct{ sous.DeploySpecs }
	// TargetManifestID is the manifest ID being targeted, after resolving all
	// context and flags.
	TargetManifestID sous.ManifestID
	// Dryrun option
	DryrunOption string
)

const (
	DryrunBoth      = DryrunOption("both")
	DryrunRegistry  = DryrunOption("registry")
	DryrunScheduler = DryrunOption("scheduler")
	DryrunNeither   = DryrunOption("none")
)

// BuildGraph builds the dependency injection graph, used to populate commands
// invoked by the user.
func BuildGraph(in io.Reader, out, err io.Writer) *SousGraph {
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
	AddFilesystem(graph)
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

// AddLogs adds a logset to the graph
func AddLogs(graph adder) {
	graph.Add(
		newLogSet,
	)
}

// AddUser adds the OS user to the graph
func AddUser(graph adder) {
	graph.Add(
		newLocalUser,
	)
}

// AddShells adds working shells to the graph
func AddShells(graph adder) {
	graph.Add(
		newLocalWorkDirShell,
		newScratchDirShell,
	)
}

// AddFilesystem adds filesystem to the graph
func AddFilesystem(graph adder) {
	graph.Add(
		newLocalSousConfig,
		newLocalWorkDir,
	)
}

// AddNetwork adds features that require the network
func AddNetwork(graph adder) {
	graph.Add(
		newDockerClient,
	)
}

// AddDocker adds Docker to the graph
func AddDocker(graph adder) {
	graph.Add(
		newDockerBuilder,
		newSelector,
	)
}

// AddSingularity adds Singularity clients to the graph
func AddSingularity(graph adder) {
	graph.Add(
		newDeployer,
	)
}

// AddState adds state reader and writers to the graph
func AddState(graph adder) {
	graph.Add(
		newStateManager,
		newLocalStateReader,
		newLocalStateWriter,
	)
}

// AddInternals adds the dependency contructors that are internal to Sous
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
		newLocalGitClient,
		newLocalGitRepo,
		newGitSourceContext,
		newSourceHostChooser,
		newCurrentState,
		newCurrentGDM,
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

func newAutoResolver(rez *sous.Resolver, sr LocalStateReader, ls *sous.LogSet) *sous.AutoResolver {
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

func newGitSourceContext(g LocalGitRepo) (GitSourceContext, error) {
	c, err := g.SourceContext()
	return GitSourceContext{c}, initErr(err, "getting local git context")
}

func newSourceContext(f *config.DeployFilterFlags, g GitSourceContext) (*sous.SourceContext, error) {
	c := g.SourceContext
	if c == nil {
		c = &sous.SourceContext{}
	}
	mid, err := newTargetManifestID(f, c)
	if err != nil {
		return nil, errors.Wrapf(err, "getting source location")
	}
	sl := sous.ManifestID(mid)
	if sl.Source.Repo != c.SourceLocation().Repo {
		// TODO: Clone the repository, and use the cloned dir as source context.
		return nil, errors.Errorf("source location %q is not the same as the remote %q",
			sl.Source.Repo, c.SourceLocation().Repo)
	}
	return c, nil
}

func newBuildContext(wd LocalWorkDirShell, c *sous.SourceContext) *sous.BuildContext {
	return &sous.BuildContext{Sh: wd.Sh, Source: *c}
}

func newBuildConfig(f *config.DeployFilterFlags, p *config.PolicyFlags, bc *sous.BuildContext) *sous.BuildConfig {
	cfg := sous.BuildConfig{
		Repo:       f.Repo,
		Offset:     f.Offset,
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
	mgr := &sous.BuildManager{
		BuildConfig: bc,
		Selector:    sl,
		Labeller:    lb,
		Registrar:   rg,
	}
	return mgr
}

func newLocalUser() (v LocalUser, err error) {
	u, err := user.Current()
	v.User = &config.User{User: u}
	return v, initErr(err, "getting current user")
}

func newLocalSousConfig(u LocalUser) (v LocalSousConfig, err error) {
	v.Config, err = newConfig(u.User.ConfigFile(), u.DefaultConfig())
	return v, initErr(err, "getting configuration")
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

func newLocalWorkDirShell(l LocalWorkDir) (v LocalWorkDirShell, err error) {
	v.Sh, err = shell.DefaultInDir(string(l))
	v.TeeEcho = os.Stdout //XXX should use a writer
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

func newDeployer(dryrun DryrunOption, r sous.Registry) sous.Deployer {
	// Eventually, based on configuration, we may make different decisions here.
	if dryrun == DryrunBoth || dryrun == DryrunScheduler {
		drc := sous.NewDummyRectificationClient(r)
		drc.SetLogger(log.New(os.Stdout, "rectify: ", 0))
		return singularity.NewDeployer(r, drc)
	}
	return singularity.NewDeployer(r, singularity.NewRectiAgent(r))
}

func newDockerClient() LocalDockerClient {
	return LocalDockerClient{docker_registry.NewClient()}
}

func newStateManager(c LocalSousConfig) (*StateManager, error) {
	if c.Server != "" {
		hsm, err := sous.NewHTTPStateManager(c.Server)
		if err != nil {
			return nil, err
		}
		return &StateManager{StateManager: hsm}, nil
	}
	dm := storage.NewDiskStateManager(c.StateLocation)
	return &StateManager{StateManager: storage.NewGitStateManager(dm)}, nil
}

func newLocalStateReader(sm *StateManager) LocalStateReader {
	return LocalStateReader{sm}
}

func newLocalStateWriter(sm *StateManager) LocalStateWriter {
	return LocalStateWriter{sm}
}

func newCurrentState(sr LocalStateReader) (*sous.State, error) {
	state, err := sr.ReadState()
	if os.IsNotExist(err) {
		log.Println("error reading state:", err)
		log.Println("defaulting to empty state")
		return sous.NewState(), nil
	}
	return state, initErr(err, "reading sous state")
}

func newCurrentGDM(state *sous.State) (CurrentGDM, error) {
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
	return docker.NewNameCache(cl.Client, db), nil
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
