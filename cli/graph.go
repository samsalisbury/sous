package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"

	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/ext/git"
	"github.com/opentable/sous/ext/singularity"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/shell"
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
	Version struct{ semv.Version }
	// LocalUser is the currently logged in user.
	LocalUser struct{ *User }
	// LocalSousConfig is the configuration for Sous.
	LocalSousConfig struct{ *sous.Config }
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
)

// BuildGraph builds the dependency injection graph, used to populate commands
// invoked by the user.
func BuildGraph(s *Sous, c *cmdr.CLI) (*SousCLIGraph, error) {
	g := &SousCLIGraph{psyringe.New()}
	return g, g.Fill(
		s, c,
		newOut,
		newErrOut,
		newLocalUser,
		newLocalSousConfig,
		newLocalWorkDir,
		newLocalWorkDirShell,
		newScratchDirShell,
		newLocalGitClient,
		newLocalGitRepo,
		newSourceContext,
		newBuildContext,
		newDockerClient,
		newBuilder,
		newDeployer,
		newRegistry,
	)
}

func newOut(c *cmdr.CLI) Out {
	return Out{c.Out}
}

func newErrOut(c *cmdr.CLI) ErrOut {
	return ErrOut{c.Err}
}

func newSourceContext(g LocalGitRepo) (c *sous.SourceContext, err error) {
	c, err = g.SourceContext()
	return c, initErr(err, "getting local git context")
}

func newBuildContext(wd LocalWorkDirShell, c *sous.SourceContext) *sous.BuildContext {
	return &sous.BuildContext{
		Sh:     wd.Sh,
		Source: *c,
	}
}

func newLocalWorkDir() (LocalWorkDir, error) {
	s, err := os.Getwd()
	return LocalWorkDir(s), initErr(err, "determining working directory")
}

func newLocalUser() (v LocalUser, err error) {
	u, err := user.Current()
	v.User = &User{u}
	return v, initErr(err, "getting current user")
}

func newLocalSousConfig(u LocalUser) (v LocalSousConfig, err error) {
	v.Config, err = newConfig(u.User)
	return v, initErr(err, "getting default config")
}

func newLocalWorkDirShell(l LocalWorkDir) (v LocalWorkDirShell, err error) {
	v.Sh, err = shell.DefaultInDir(string(l))
	v.TeeEcho = os.Stdout
	v.TeeOut = os.Stdout
	v.TeeErr = os.Stderr
	return v, initErr(err, "getting current working directory")
}

// TODO: This should register a cleanup task with the cli, to delete the temp
// dir.
func newScratchDirShell() (v ScratchDirShell, err error) {
	what := "getting scratch directory"
	dir, err := ioutil.TempDir("", "sous")
	if err != nil {
		return v, initErr(err, what)
	}
	v.Sh, err = shell.DefaultInDir(dir)
	v.TeeOut = os.Stdout
	v.TeeErr = os.Stderr
	return v, initErr(err, what)
}

func newLocalGitClient(sh LocalWorkDirShell) (v LocalGitClient, err error) {
	v.Client, err = git.NewClient(sh.Sh)
	return v, initErr(err, "initialising git client")
}

func newLocalGitRepo(c LocalGitClient) (v LocalGitRepo, err error) {
	v.Repo, err = c.OpenRepo(".")
	return v, initErr(err, "opening local git repository")
}

func newBuilder(cl LocalDockerClient, ctx *sous.SourceContext, source LocalWorkDirShell, scratch ScratchDirShell, u LocalUser) (sous.Builder, error) {
	return makeDockerBuilder(cl, ctx, source, scratch, u)
}

func newRegistry(cl LocalDockerClient, u LocalUser) (sous.Registry, error) {
	return makeDockerRegistry(cl, u)
}

func makeDockerRegistry(cl LocalDockerClient, u LocalUser) (*docker.NameCache, error) {
	cfg := u.DefaultConfig()
	dbCfg := &docker.DBConfig{Driver: cfg.DatabaseDriver, Connection: cfg.DatabaseConnection}
	db, err := docker.GetDatabase(dbCfg)
	if err != nil {
		return nil, fmt.Errorf("unable to build name cache DB: ", err)
	}
	return &docker.NameCache{cl.Client, db}, nil
}

func makeDockerBuilder(cl LocalDockerClient, ctx *sous.SourceContext, source LocalWorkDirShell, scratch ScratchDirShell, u LocalUser) (*docker.Builder, error) {
	nc, err := makeDockerRegistry(cl, u)
	if err != nil {
		return nil, err
	}
	// TODO: Get this from config.
	drh := "docker.otenv.com"
	return docker.NewBuilder(nc, drh, ctx, source.Sh, scratch.Sh)
}

func newDeployer(r sous.Registry) (sous.Deployer, error) {
	ra := singularity.NewRectiAgent(r)
	return singularity.NewDeployer(r, ra), nil
}

func newDockerClient() LocalDockerClient {
	return LocalDockerClient{docker_registry.NewClient()}
}

// initErr returns nil if error is nil, otherwise an initialisation error.
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
