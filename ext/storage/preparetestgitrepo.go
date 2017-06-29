package storage

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	sous "github.com/opentable/sous/lib"
)

var testUser = sous.User{Name: "Test User", Email: "test@user.com"}

func PrepareTestGitRepo(t *testing.T, s *sous.State, remotepath, outpath string) {

	clobberDir(t, remotepath)
	clobberDir(t, outpath)

	runCmd(t, remotepath, "git", "init", "--template=/dev/null", "--bare")

	remoteAbs, err := filepath.Abs(remotepath)
	if err != nil {
		t.Fatal(err)
	}

	runCmd(t, outpath, "git", "init", "--template=/dev/null")
	runCmd(t, outpath, "git", "config", "user.email", "sous-test@testing.example.com")
	runCmd(t, outpath, "git", "config", "user.name", "sous-test@testing.example.com")
	runCmd(t, outpath, "git", "remote", "add", "origin", "file://"+remoteAbs)

	dsm := NewDiskStateManager(outpath)
	dsm.WriteState(s, testUser)

	runCmd(t, outpath, "git", "add", ".")
	runCmd(t, outpath, "git", "commit", "--no-gpg-sign", "-a", "-m", "birthday")
	runCmd(t, outpath, "git", "push", "-u", "origin", "master")
}

func clobberDir(t *testing.T, path string) {
	if err := os.RemoveAll(path); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		t.Fatal(err)
	}
}

func runCmd(t *testing.T, path string, cmd ...string) {
	gitCmd := exec.Command(cmd[0], cmd[1:]...)
	gitCmd.Dir = path
	out, err := gitCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%q errored: %v\n %s", strings.Join(cmd, " "), err, out)
	}
}
