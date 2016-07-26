// Package storage is responsible for the persistent storage of state.
//
// Sous state is stored in a file hierarchy like this:
//
//     /
//         defs.yaml
//         manifests/
//             github.com/
//                 username/
//                     reponame/
//                         dirname/
//                             subdirname.yaml
package storage

import (
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/hy"
)

type (
	// StateReader knows how to read state.
	StateReader interface {
		ReadState() (*sous.State, error)
	}
	// StateWriter know how to write state.
	StateWriter interface {
		WriteState(*sous.State) error
	}
	// DiskStateManager implements StateReader and StateWriter using disk
	// storage as its back-end.
	DiskStateManager struct {
		baseDir string
	}
)

func NewDiskStateManager(baseDir string) (*DiskStateManager, error) {
	return &DiskStateManager{baseDir}, nil
}

// ReadState loads the entire intended state of the world from a dir.
func (dsm *DiskStateManager) ReadState() (*sous.State, error) {
	s := &sous.State{}
	return s, hy.Unmarshal(dsm.baseDir, s)
}

// WriteState records the entire intended state of the world to a dir.
func (dsm *DiskStateManager) WriteState(s *sous.State) error {
	return hy.Marshal(dsm.baseDir, s)
}
