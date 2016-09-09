package storage

import (
	"github.com/opentable/sous/ext/git"
	"github.com/opentable/sous/lib"
)

// GitStateManager wraps a DiskStateManager and implements transactional writes
// to a Git remote. It also polls the Git remote for changes
//
// Methods of GitStateManager are serialised, and thus safe for concurrent
// access. No two GitStateManagers should have DiskStateManagers using the same
// BaseDir.
type GitStateManager struct {
}

// NewGitStateManager creates a new GitStateManager wrapping the provided
// DiskStateManager.
func NewGitStateManager(dsm DiskStateManager, remote git.Remote) {

}

// ReadState reads sous state from the local disk.
func (gsm *GitStateManager) ReadState() (*sous.State, error) {
	panic(nil)
}

// WriteState writes sous state to disk, then attempts to push it to Remote.
// If the push fails, the state is reset and an error is returned.
func (gsm *GitStateManager) WriteState(*sous.State) error {
	panic(nil)
}
