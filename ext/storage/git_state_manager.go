package storage

import (
	"log"
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
	//git.Env = []string{"GIT_CONFIG_NOSYSTEM=true", "HOME=none", "XDG_CONFIG_HOME=none"}
	//log.Print(git)
	out, err := git.CombinedOutput()
	return errors.Wrapf(err, strings.Join(git.Args, " ")+": "+string(out))
}

func (gsm *GitStateManager) revert(tn string) {
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

// WriteState writes sous state to disk, then attempts to push it to Remote.
// If the push fails, the state is reset and an error is returned.
func (gsm *GitStateManager) WriteState(s *sous.State) error {
	// git pull

	tn := "sous-fallback-" + uuid.New()
	if err := gsm.git("tag", tn); err != nil {
		return err
	}
	defer gsm.git("tag", "-d", tn)

	if err := gsm.DiskStateManager.WriteState(s); err != nil {
		return err
	}
	if err := gsm.git(`add`, `.`); err != nil {
		gsm.revert(tn)
		return err
	}
	if err := gsm.git(`commit`, `-m`, `""`); err != nil {
		gsm.revert(tn)
		return err
	}
	err := gsm.git(`push`)
	if err != nil {
		gsm.revert(tn)
	}
	log.Print(err)
	return err

	// git commit -a -m ""
	// git push
	// Problems?
	//   git checkout tag
	// git tag -d <t>
}
