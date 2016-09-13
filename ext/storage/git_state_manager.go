package storage

import (
	"sync"

	"github.com/opentable/sous/lib"
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

// ReadState reads sous state from the local disk.
func (gsm *GitStateManager) ReadState() (*sous.State, error) {
	// git pull
	return gsm.DiskStateManager.ReadState()
}

// WriteState writes sous state to disk, then attempts to push it to Remote.
// If the push fails, the state is reset and an error is returned.
func (gsm *GitStateManager) WriteState(s *sous.State) error {
	// git pull
	// git tag <t>
	return gsm.DiskStateManager.WriteState(s)
	// git commit -a -m ""
	// git push
	// Problems?
	//   git checkout tag
	// git tag -d <t>
}
