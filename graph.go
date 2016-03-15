package main

import (
	"fmt"
	"os"
	"os/user"

	"github.com/opentable/sous2/ext/git"
	"github.com/opentable/sous2/util/shell"
	"github.com/samsalisbury/psyringe"
)

type (
	// LocalUser is the logged in user who invoked Sous
	LocalUser *user.User
	// LocalSousConfig is the configuration for Sous.
	LocalSousConfig Config
	// WorkDir is the user's current working directory when they invoke Sous.
	LocalWorkDir string
	// WorkdirShell is a shell for working in the user's current working
	// directory.
	LocalWorkDirShell *shell.Sh
	// LocalGitClient is a git client rooted in WorkdirShell.Dir.
	LocalGitClient *git.Client
	// LocalGitRepo is the git repository containing WorkDir.
	LocalGitRepo *git.Repo
	// LocalGitContext is the git context snapshot of the user when they invok
	// Sous.
	LocalGitContext *git.Context
	// ScratchDirShell is a shell for working in the scratch area where things
	// like artifacts, and build metadata are stored.
	ScratchDirShell *shell.Sh
)

func initError(what string, err error) error {
	message := fmt.Sprintf("error initialising %s:", what)
	if shellErr, ok := err.(shell.Error); ok {
		message += fmt.Sprintf("\ncommand failed:\nshell> %s\n%s",
			shellErr.Command.String(), shellErr.Result.Combined.String())
	} else {
		message += err.Error()
	}
	return fmt.Errorf(message)
}

func buildDeps() (*psyringe.Psyringe, error) {
	s := psyringe.New()
	err := s.Fill(
		newLocalUser,
		newLocalSousConfig,
		newLocalWorkDir,
		newLocalWorkDirShell,
		newScratchDirShell,
		newLocalGitClient,
		newLocalGitRepo,
		newLocalGitContext,
	)
	return s, err
}

func newLocalWorkDir() (LocalWorkDir, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", initError("determine working directory", err)
	}
	return LocalWorkDir(wd), nil
}

func newLocalUser() (LocalUser, error) {
	u, err := user.Current()
	if err != nil {
		return nil, initError("get current user", err)
	}
	return u, nil
}

func newLocalSousConfig(u LocalUser) (LocalSousConfig, error) {
	c, err := newDefaultConfig(u)
	if err != nil {
		return LocalSousConfig{}, initError("get default config", err)
	}
	return LocalSousConfig(c), nil
}

func newLocalWorkDirShell(LocalWorkDir) (LocalWorkDirShell, error) {
	s, err := shell.Default()
	if err != nil {
		return nil, initError("get current working directory", err)
	}
	return s, nil
}

func newScratchDirShell(c LocalSousConfig) (ScratchDirShell, error) {
	const what = "get scratch directory"
	s, err := shell.Default()
	if err != nil {
		return nil, initError(what, err)
	}
	if err := s.CD(c.SousSettingsDir); err != nil {
		return nil, initError(what, err)
	}
	return s, nil
}

func newLocalGitClient(sh LocalWorkDirShell) (LocalGitClient, error) {
	c, err := git.NewClient(sh)
	if err != nil {
		return nil, initError("initialising git client", err)
	}
	return c, nil
}

func newLocalGitRepo(c LocalGitClient) (LocalGitRepo, error) {
	r, err := (*git.Client)(c).OpenRepo(".")
	if err != nil {
		return nil, initError("opening local git repository", err)
	}
	return r, nil
}

func newLocalGitContext(r LocalGitRepo) (LocalGitContext, error) {
	c, err := (*git.Repo)(r).Context()
	if err != nil {
		return nil, initError("getting git context", err)
	}
	return c, nil
}
