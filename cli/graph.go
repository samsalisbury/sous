package cli

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"

	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/git"
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
	// SousCLIGraph is a dependency injector used to flesh out Sous commands
	// with their dependencies.
	SousCLIGraph struct{ *psyringe.Psyringe }
	// Version represents a version of Sous.

	// OutWriter is an alias on io.Writer to disguish "stderr"
	OutWriter io.Writer
	// ErrWriter is an alias on io.Writer to disguish "stderr"
	ErrWriter io.Writer
)

type (
	Version struct{ semv.Version }
	// LocalUser is the currently logged in user.
	LocalUser struct{ *User }
	// LocalSousConfig is the configuration for Sous.
	LocalSousConfig struct{ *Config }
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
	// LocalStateReader wraps a storage.StateReader, and should be configured
	// to use the current user's local storage.
	LocalStateReader struct{ storage.StateReader }
	// LocalStateWriter wraps a storage.StateWriter, and should be configured to
	// use the current user's local storage.
	LocalStateWriter struct{ storage.StateWriter }
	// CurrentGDM is a snapshot of the GDM at application start. In a CLI
	// context, which this is, that is all we need to simply read the GDM.
	CurrentGDM struct{ *sous.State }
)

// BuildGraph builds the dependency injection graph, used to populate commands
// invoked by the user.
func BuildGraph(c *cmdr.CLI, out, err io.Writer) *SousCLIGraph {
	return &SousCLIGraph{psyringe.New(
		c,
		func() OutWriter { return out },
		func() ErrWriter { return err },
		newOut,
		newErrOut,
		newLogSet,
		newLocalUser,
		newLocalSousConfig,
		newLocalWorkDir,
		newLocalWorkDirShell,
		newScratchDirShell,
		newLocalGitClient,
		newLocalGitRepo,
		newGitSourceContext,
		newSourceContext,
		newBuildContext,
		newDockerClient,
		newDockerBuilder,
		newSelector,
		newLabeller,
		newRegistrar,
		newDeployer,
		newRegistry,
		newLocalDiskStateManager,
		newLocalStateReader,
		newLocalStateWriter,
		newCurrentGDM,
	)}
}

func newOut(c *cmdr.CLI) Out {
	return Out{c.Out}
}

func newErrOut(c *cmdr.CLI) ErrOut {
	return ErrOut{c.Err}
}

func newLogSet(s *Sous, err ErrWriter) *sous.LogSet { // XXX temporary until we settle on logging
	fmt.Println(s.flags.Verbosity)
	if s.flags.Verbosity.Debug {
		fmt.Println("debug level")
		if s.flags.Verbosity.Loud {
			sous.Log.Vomit.SetOutput(err)
		}
		sous.Log.Debug.SetOutput(err)
		sous.Log.Info.SetOutput(err)

	}
	if s.flags.Verbosity.Loud {
		sous.Log.Info.SetOutput(err)
	}
	if s.flags.Verbosity.Quiet {
	}
	if s.flags.Verbosity.Silent {
	}

	sous.Log.Vomit.Println("Verbose debugging enabled")
	sous.Log.Debug.Println("Regular debugging enabled")
	sous.Log.Info.Println("Informational messages enabled")
	return &sous.Log
}

/*
func newSourceFlags(c *cmdr.CLI) (*DeployFilterFlags, error) {
	sourceFlags := &DeployFilterFlags{}
	var err error
	c.AddGlobalFlagSetFunc(func(fs *flag.FlagSet) {
		err = AddFlags(fs, sourceFlags, sourceFlagsHelp)
		if err != nil {
			panic(err)
		}
	})
	return sourceFlags, err
}
*/

func newGitSourceContext(g LocalGitRepo) (GitSourceContext, error) {
	c, err := g.SourceContext()
	return GitSourceContext{c}, initErr(err, "getting local git context")
}

func newSourceContext(g GitSourceContext, f *DeployFilterFlags) (*sous.SourceContext, error) {
	c := g.SourceContext
	if c == nil {
		c = &sous.SourceContext{}
	}

	sl, err := resolveSourceLocation(f, c)
	if err != nil {
		return nil, errors.Wrap(err, "resolving source location")
	}
	if sl != c.SourceLocation() {
		// TODO: Clone the repository, and use the cloned dir as source context.
		return nil, errors.Errorf("source location %q is not the same as the remote %q",
			sl, c.SourceLocation())
	}
	return c, nil
}

func newBuildContext(wd LocalWorkDirShell, c *sous.SourceContext) *sous.BuildContext {
	return &sous.BuildContext{Sh: wd.Sh, Source: *c}
}

func newLocalUser() (v LocalUser, err error) {
	u, err := user.Current()
	v.User = &User{u}
	return v, initErr(err, "getting current user")
}

func newLocalSousConfig(u LocalUser) (v LocalSousConfig, err error) {
	v.Config, err = newConfig(u.User)
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
	v.TeeEcho = os.Stdout
	v.TeeOut = os.Stdout
	v.TeeErr = os.Stderr
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

func newRegistry(cfg LocalSousConfig, cl LocalDockerClient) (sous.Registry, error) {
	return makeDockerRegistry(cfg, cl)
}
func newDeployer(r sous.Registry) sous.Deployer {
	// Eventually, based on configuration, we may make different decisions here.
	return singularity.NewDeployer(r, singularity.NewRectiAgent(r))
}

func newDockerClient() LocalDockerClient {
	return LocalDockerClient{docker_registry.NewClient()}
}

func newLocalDiskStateManager(c LocalSousConfig) (*storage.DiskStateManager, error) {
	sm, err := storage.NewDiskStateManager(c.StateLocation)
	return sm, initErr(err, "initialising sous state")
}

func newLocalStateReader(sm *storage.DiskStateManager) LocalStateReader {
	return LocalStateReader{sm}
}

func newLocalStateWriter(sm *storage.DiskStateManager) LocalStateWriter {
	return LocalStateWriter{sm}
}

func newCurrentGDM(sr LocalStateReader) (CurrentGDM, error) {
	gdm, err := sr.ReadState()
	if os.IsNotExist(err) {
		log.Printf("error reading state: %s", err)
		log.Println("defaulting to empty state")
		return CurrentGDM{&sous.State{}}, nil
	}
	return CurrentGDM{gdm}, initErr(err, "reading sous state")
}

// The funcs named makeXXX below are used to create specific implementations of
// sous native types.

// makeDockerRegistry creates a Docker version of sous.Registry
func makeDockerRegistry(cfg LocalSousConfig, cl LocalDockerClient) (*docker.NameCache, error) {
	dbCfg := cfg.Docker.DBConfig()
	db, err := docker.GetDatabase(&dbCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to build name cache DB: %s", err)
	}
	return &docker.NameCache{RegistryClient: cl.Client, DB: db}, nil
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
