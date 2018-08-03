package smoke

import (
	"path/filepath"
	"testing"
)

type gitClient struct {
	Bin
}

func newGitClient(t *testing.T, f *fixture, name string) *gitClient {
	t.Helper()
	baseDir := filepath.Join(f.BaseDir, name)
	bin := NewBin(t, "git", "gitclient1", baseDir, f.Finished)
	addGitEnvVars(bin.Env)
	return &gitClient{
		Bin: bin,
	}
}

type gitRepoSpec struct {
	UserName, UserEmail, OriginURL string
}

func (g *gitClient) configureRepo(t *testing.T, f *fixture, dir string, spec gitRepoSpec) string {
	t.Helper()
	g.Bin.Configure()
	g.Bin.Dir = makeEmptyDir(f.BaseDir, dir)

	// Ensure we cannot see a global git config file.
	g.MustFail(t, "config", nil, "--global", "-l")
	g.MustRun(t, "init", nil)

	// TODO SS: Speed this up by just writing config file once?
	g.MustRun(t, "remote", nil, "add", "origin", spec.OriginURL)
	g.MustRun(t, "config", nil, "user.name", spec.UserName)
	g.MustRun(t, "config", nil, "user.email", spec.UserEmail)
	g.MustRun(t, "config", nil, "commit.gpgsign", "false")

	// Dump config to logs.
	g.MustRun(t, "config", nil, "-l")

	return g.Bin.Dir
}
