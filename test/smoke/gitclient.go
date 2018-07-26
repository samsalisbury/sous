package smoke

import "testing"

type gitClient struct {
	Bin
}

func newGitClient(t *testing.T, f *testFixture, baseDir string) *gitClient {
	bin := NewBin(t, "git", "gitclient1", baseDir, f.Finished)
	addGitEnvVars(bin.Env)
	return &gitClient{
		Bin: bin,
	}
}

type gitRepoSpec struct {
	UserName, UserEmail, OriginURL string
}

func (g *gitClient) configureRepo(t *testing.T, dir string, spec gitRepoSpec) string {
	g.Bin.Configure()
	g.Bin.Dir = makeEmptyDir(t, g.Bin.BaseDir, dir)
	g.MustRun(t, "init", nil)
	g.MustRun(t, "remote", nil, "add", "origin", spec.OriginURL)
	g.MustRun(t, "config", nil, "user.name", spec.UserName)
	g.MustRun(t, "config", nil, "user.email", spec.UserEmail)
	return g.Bin.Dir
}
