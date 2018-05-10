package storage

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
)

// GitStateManager wraps a DiskStateManager and implements transactional writes
// to a Git remote. It also polls the Git remote for changes
//
// Methods of GitStateManager are serialised, and thus safe for concurrent
// access. No two GitStateManagers should have DiskStateManagers using the same
// BaseDir.
type (
	GitStateManager struct {
		// All reads and writes must use exclusive lock, because read affects state
		// by doing a git pull.
		sync.Mutex
		*DiskStateManager //can't just be a StateReader/Writer: needs dir
		remote            string
	}

	gsmError string
)

func (err gsmError) Error() string {
	return string(err)
}

// IsGSMError returns true if err is a git-state-manager error
func IsGSMError(err error) bool {
	_, is := err.(gsmError)
	return is
}

// NewGitStateManager creates a new GitStateManager wrapping the provided
// DiskStateManager.
func NewGitStateManager(dsm *DiskStateManager) *GitStateManager {
	return &GitStateManager{DiskStateManager: dsm}
}

func (gsm *GitStateManager) git(cmd ...string) error {
	_, err := gsm.gitOut(cmd...)
	return err
}

func (gsm *GitStateManager) log() logging.LogSink {
	return *(logging.SilentLogSet().Child("GitStateManager").(*logging.LogSet))
}

func (gsm *GitStateManager) gitOut(cmd ...string) (string, error) {
	if !gsm.isRepo() {
		return "", gsmError("not in a git repo")
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
		messages.ReportLogFieldsMessage("success", logging.DebugLevel, gsm.log(), git.Args)
	} else {
		messages.ReportLogFieldsMessage("error", logging.DebugLevel, gsm.log(), err)
	}
	messages.ReportLogFieldsMessage("git", logging.ExtraDebug1Level, gsm.log(), string(out))
	return string(out), errors.Wrapf(err, strings.Join(git.Args, " ")+": "+string(out))
}

func (gsm *GitStateManager) reset(tn string) {
	gsm.git("reset", "--hard", tn)
	gsm.git("clean", "-f")
}

func (gsm *GitStateManager) isRepo() bool {
	s, err := os.Stat(filepath.Join(gsm.DiskStateManager.BaseDir, ".git"))
	return err == nil && s.IsDir()
}

func (gsm *GitStateManager) headRev() (string, error) {
	etag, err := gsm.gitOut("rev-parse", "HEAD")
	if err != nil {
		// XXX Is this the right thing to do?
		return "", err
	}

	return strings.TrimSpace(etag), nil
}

// ReadState reads sous state from the local disk.
func (gsm *GitStateManager) ReadState() (*sous.State, error) {
	gsm.Lock()
	defer gsm.Unlock()
	// git pull
	gsm.git("pull")

	state, err := gsm.DiskStateManager.ReadState()
	if err != nil {
		return state, err
	}

	etag, err := gsm.headRev()
	if err != nil {
		// XXX Is this the right thing to do?
		return state, err
	}

	state.SetEtag(etag)
	return state, nil
}

func (gsm *GitStateManager) needCommit() bool {
	err := gsm.git("diff-index", "--exit-code", "HEAD")
	if ee, is := errors.Cause(err).(*exec.ExitError); is {
		return !ee.Success()
	}
	return false
}

func (gsm *GitStateManager) assertOneChange() error {
	diffIndex, err := gsm.gitOut("diff-index", "--cached", "master@{upstream}")
	if err != nil {
		return err
	}

	firstNL := strings.IndexByte(diffIndex, '\n')
	if firstNL == -1 {
		// empty diff-index?
		return nil
	}

	secondNL := strings.IndexByte(diffIndex[firstNL+1:], '\n')
	if secondNL != -1 {
		return errors.Errorf("git update touches more than one file: %q", diffIndex)
	}

	return nil
}

// WriteState writes sous state to disk, then attempts to push it to Remote.
// If the push fails, the state is reset and an error is returned.
func (gsm *GitStateManager) WriteState(s *sous.State, u sous.User) error {
	gsm.Lock()
	defer gsm.Unlock()

	etag, err := gsm.headRev()
	if err != nil {
		return err
	}
	if err := s.CheckEtag(etag); err != nil {
		return err
	}

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
		err := gsm.assertOneChange()
		if err != nil {
			gsm.reset(tn)
			return err
		}

		err = gsm.git("push", "-u", "origin", "master")
		if err == nil {
			// Success.
			return nil
		}
		messages.ReportLogFieldsMessage("git push failed; trying again with # attempts left", logging.DebugLevel, gsm.log(), remainingAttempts, err)
		gsm.reset(tn)
		if err := gsm.git("pull"); err != nil {
			return err
		}
		if err := gsm.git("cherry-pick", newTag); err != nil {
			// If cherry-pick fails, then there's a real conflict.
			messages.ReportLogFieldsMessage("attempt to rectify conflicts with git cherry-pick failed", logging.WarningLevel, gsm.log(), err)
			if err := gsm.git("cherry-pick", "--abort"); err != nil {
				messages.ReportLogFieldsMessage("cherry-pick --abort failed", logging.WarningLevel, gsm.log(), err)
				return err
			}
			messages.ReportLogFieldsMessage("Successfully cherry-picked new changes, re-attempting push.", logging.DebugLevel, gsm.log())
		}
	}
	return fmt.Errorf("unable to merge changes")
}
