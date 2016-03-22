package cli

import (
	"fmt"
	"os"
	"os/user"

	"github.com/opentable/sous/ext/git"
	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/util/shell"
	"github.com/samsalisbury/psyringe"
	"github.com/samsalisbury/semv"
)

type (
	// Out is an output used for real data a Command returns. This should only
	// be used when a command needs to write directly to stdout, using the
	// formatting options that come with an output. Usually, you should use a
	// SuccessResult with Data to return data.
	Out struct{ Output }
	// ErrOut is an output used for logging from a Command. This should only be
	// used when a Command needs to write a lot of data to stderr, using the
	// formatting options that come with and Output. Usually you should use and
	// ErrorResult to return error messages.
	ErrOut struct{ Output }
	// SousCLIGraph is a dependency injector used to flesh out Sous commands
	// with their dependencies.
	SousCLIGraph struct{ *psyringe.Psyringe }
	// SousVersion represents a version of Sous.
	Version struct{ semv.Version }
	// LocalUser is the logged in user who invoked Sous
	LocalUser struct{ *user.User }
	// LocalSousConfig is the configuration for Sous.
	LocalSousConfig struct{ sous.Config }
	// WorkDir is the user's current working directory when they invoke Sous.
	LocalWorkDir string
	// WorkdirShell is a shell for working in the user's current working
	// directory.
	LocalWorkDirShell struct{ *shell.Sh }
	// LocalGitClient is a git client rooted in WorkdirShell.Dir.
	LocalGitClient struct{ *git.Client }
	// LocalGitRepo is the git repository containing WorkDir.
	LocalGitRepo struct{ *git.Repo }
	// LocalGitContext is the git context snapshot of the user when they invok
	// Sous.
	LocalGitContext struct{ *git.Context }
	// ScratchDirShell is a shell for working in the scratch area where things
	// like artifacts, and build metadata are stored.
	ScratchDirShell struct{ *shell.Sh }
)

// buildGraph builds the dependency injection graph, used to populate commands
// invoked by the user.
func BuildGraph(c *CLI, s *Sous) (*SousCLIGraph, error) {
	g := &SousCLIGraph{psyringe.New()}
	return g, g.Fill(
		c, s,
		newOut,
		newErrOut,
		newLocalUser,
		newLocalSousConfig,
		newLocalWorkDir,
		newLocalWorkDirShell,
		newScratchDirShell,
		newLocalGitClient,
		newLocalGitRepo,
		newLocalGitContext,
	)
}

func newOut(c *CLI) Out {
	return Out{c.out}
}

func newErrOut(c *CLI) ErrOut {
	return ErrOut{c.err}
}

func newLocalWorkDir() (LocalWorkDir, error) {
	s, err := os.Getwd()
	return LocalWorkDir(s), initErr(err, "determining working directory")
}

func newLocalUser() (v LocalUser, err error) {
	v.User, err = user.Current()
	return v, initErr(err, "getting current user")
}

func newLocalSousConfig(u LocalUser) (v LocalSousConfig, err error) {
	v.Config, err = newDefaultConfig(u.User)
	return v, initErr(err, "getting default config")
}

func newLocalWorkDirShell(l LocalWorkDir) (v LocalWorkDirShell, err error) {
	v.Sh, err = shell.DefaultInDir(string(l))
	return v, initErr(err, "getting current working directory")
}

func newScratchDirShell(c LocalSousConfig) (v ScratchDirShell, err error) {
	v.Sh, err = shell.DefaultInDir(c.SettingsDir)
	return v, initErr(err, "getting scratch directory")
}

func newLocalGitClient(sh LocalWorkDirShell) (v LocalGitClient, err error) {
	v.Client, err = git.NewClient(sh.Sh)
	return v, initErr(err, "initialising git client")
}

func newLocalGitRepo(c LocalGitClient) (v LocalGitRepo, err error) {
	v.Repo, err = c.OpenRepo(".")
	return v, initErr(err, "opening local git repository")
}

func newLocalGitContext(r LocalGitRepo) (v LocalGitContext, err error) {
	v.Context, err = r.Context()
	return v, initErr(err, "getting git context")
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
		message += err.Error()
	}
	return fmt.Errorf(message)
}
