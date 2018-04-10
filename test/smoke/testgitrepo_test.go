//+build smoke

package smoke

import "testing"

type GitRepoSpec struct {
	UserName, UserEmail, OriginURL string
}

func makeGitRepo(t *testing.T, baseDir, dir string, spec GitRepoSpec) string {
	dir = makeEmptyDir(t, baseDir, dir)
	if err := doCMD(dir, "git", "init"); err != nil {
		t.Fatalf("git init failed in %q: %s", dir, err)
	}
	if err := doCMD(dir, "git", "remote", "add", "origin", "git@github.com:opentable/bogus/repo.git"); err != nil {
		t.Fatalf("git remote add failed in %q: %s", dir, err)
	}
	if err := doCMD(dir, "git", "config", "user.name", "Sous User"); err != nil {
		t.Fatalf("git config failed in %q: %s", dir, err)
	}
	if err := doCMD(dir, "git", "config", "user.email", "sous-user@example.com"); err != nil {
		t.Fatalf("git config failed in %q: %s", dir, err)
	}
	return dir
}
