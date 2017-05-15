package storage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/opentable/sous/lib"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

// GitStateManager wraps a DiskStateManager and implements transactional writes
// to a Git remote. It also polls the Git remote for changes
//
// Methods of GitStateManager are serialised, and thus safe for concurrent
// access. No two GitStateManagers should have DiskStateManagers using the same
// BaseDir.
type GitStateManager struct {
	sync.Mutex
	*DiskStateManager //can't just be a StateReader/Writer: needs dir
	remote            string
}

// NewGitStateManager creates a new GitStateManager wrapping the provided
// DiskStateManager.
func NewGitStateManager(dsm *DiskStateManager) *GitStateManager {
	return &GitStateManager{DiskStateManager: dsm}
}

func (gsm *GitStateManager) git(cmd ...string) error {
	if !gsm.isRepo() {
		return nil
	}
	git := exec.Command(`git`, cmd...)
	git.Dir = gsm.DiskStateManager.BaseDir

	// This had been commented out, with a comment to the effect that it was fixing a bug.
	// However, it could lead to come confusion because the behavior of Sous
	// server would change based on the configuration of git outside of the GDM
	// repo.
	// It's also definitely causing problems in testing for me (JL) because my
	// local git configuration effects testing behaviors.
	git.Env = []string{"GIT_CONFIG_NOSYSTEM=true", "GIT_CONFIG_NOGLOBAL=true", "HOME=none", "XDG_CONFIG_HOME=none"}
	gitssh := os.Getenv("GIT_SSH")
	if gitssh != "" {
		git.Env = append(git.Env, "GIT_SSH="+gitssh)
	}
	out, err := git.CombinedOutput()
	if err == nil {
		sous.Log.Debug.Printf("%+v: success", git.Args)
	} else {
		sous.Log.Debug.Printf("%+v: error: %v", git.Args, err)
	}
	sous.Log.Vomit.Print("git: " + string(out))
	return errors.Wrapf(err, strings.Join(git.Args, " ")+": "+string(out))
}

func (gsm *GitStateManager) reset(tn string) {
	gsm.git("reset", "--hard", tn)
	gsm.git("clean", "-f")
}

func (gsm *GitStateManager) isRepo() bool {
	s, err := os.Stat(filepath.Join(gsm.DiskStateManager.BaseDir, ".git"))
	return err == nil && s.IsDir()
}

// ReadState reads sous state from the local disk.
func (gsm *GitStateManager) ReadState() (*sous.State, error) {
	// git pull
	gsm.git("pull")

	return gsm.DiskStateManager.ReadState()
}

func (gsm *GitStateManager) needCommit() bool {
	err := gsm.git("diff-index", "--exit-code", "HEAD")
	if ee, is := errors.Cause(err).(*exec.ExitError); is {
		return !ee.Success()
	}
	return false
}

// WriteState writes sous state to disk, then attempts to push it to Remote.
// If the push fails, the state is reset and an error is returned.
func (gsm *GitStateManager) WriteState(s *sous.State, u sous.User) error {
	tn := "sous-fallback-" + uuid.New()
	if err := gsm.git("tag", tn); err != nil {
		return err
	}
	defer gsm.git("tag", "-d", tn)

	if err := gsm.DiskStateManager.WriteState(s, u); err != nil {
		return err
	}
	if err := gsm.git(`add`, `.`); err != nil {
		gsm.reset(tn)
		return err
	}
	if !gsm.needCommit() {
		return nil
	}

	// Commit the changes.
	commitCommand := []string{"commit", "-m", "sous commit: Update State"}
	if u.Complete() {
		author := u.String()
		commitCommand = append(commitCommand, "--author", author)
	}
	if err := gsm.git(commitCommand...); err != nil {
		gsm.reset(tn)
		return err
	}

	// Tag this commit.
	newTag := "sous-new-" + uuid.New()
	if err := gsm.git("tag", newTag); err != nil {
		return err
	}
	defer gsm.git("tag", "-d", newTag)

	// If push fails:
	//   - Reset to HEAD^
	//   - Git pull (if this fails, give up)
	//   - Cherry-pick sous-new-{UUID}"
	//   - Try again.

	const gitRectifyAttempts = 5
	for remainingAttempts := gitRectifyAttempts; remainingAttempts > 0; remainingAttempts-- {
		err := gsm.git("push", "-u", "origin", "master")
		if err == nil {
			// Success.
			return nil
		}
		sous.Log.Debug.Printf("git push failed; trying again (%d attempts left): %s", remainingAttempts, err)
		gsm.reset(tn)
		if err := gsm.git("pull"); err != nil {
			return err
		}
		if err := gsm.git("cherry-pick", newTag); err != nil {
			// If cherry-pick fails, then there's a real conflict.
			sous.Log.Warn.Printf("attempt to rectify conflicts with git cherry-pick failed: %s", err)
			if err := gsm.git("cherry-pick", "--abort"); err != nil {
				sous.Log.Warn.Printf("cherry-pick --abort failed: %s", err)
				return err
			}
			sous.Log.Debug.Printf("Successfully cherry-picked new changes, re-attempting push.")
		}
	}
	return fmt.Errorf("unable to merge changes")
}
