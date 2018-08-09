package smoke

import (
	"testing"
)

type gitClient struct {
	Bin
}

func newGitClient(t *testing.T, f fixtureConfig, name string) *gitClient {
	t.Helper()
	bin := f.newBin(t, "git", name)
	addGitEnvVars(bin.Env)
	return &gitClient{
		Bin: bin,
	}
}

type gitRepoSpec struct {
	UserName, UserEmail, OriginURL string
}

func (g *gitClient) init(t *testing.T, f fixtureConfig, spec gitRepoSpec) {
	t.Helper()
	g.Bin.Configure()

	g.MustRun(t, "init", nil)
	g.MustRun(t, "remote", nil, "add", "origin", spec.OriginURL)
	g.configRepo(t, spec)
}

func (g *gitClient) cloneIntoCurrentDir(t *testing.T, f fixtureConfig, spec gitRepoSpec) {
	t.Helper()
	g.Bin.Configure()

	g.MustRun(t, "clone", nil, spec.OriginURL, ".")
	g.configRepo(t, spec)
}

func (g *gitClient) configRepo(t *testing.T, spec gitRepoSpec) {
	// Ensure we cannot see a global git config file.
	g.MustFail(t, "config", nil, "--global", "-l")

	// TODO SS: Speed this up by just writing config file once?
	g.MustRun(t, "config", nil, "user.name", spec.UserName)
	g.MustRun(t, "config", nil, "user.email", spec.UserEmail)
	g.MustRun(t, "config", nil, "commit.gpgsign", "false")

	// Dump config to logs.
	g.MustRun(t, "config", nil, "-l")
}
