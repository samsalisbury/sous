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
	"github.com/opentable/hy"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/yaml"
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
		Codec   *hy.Codec
	}
)

// NewDiskStateManager returns a new DiskStateManager configured to read and
// write from a filesystem tree containing YAML files.
func NewDiskStateManager(baseDir string) *DiskStateManager {
	c := hy.NewCodec(func(c *hy.Codec) {
		c.FileExtension = "yaml"
		c.MarshalFunc = yaml.Marshal
		c.UnmarshalFunc = yaml.Unmarshal
	})
	return &DiskStateManager{Codec: c, baseDir: baseDir}
}

// ReadState loads the entire intended state of the world from a dir.
func (dsm *DiskStateManager) ReadState() (*sous.State, error) {
	s := &sous.State{}
	return s, dsm.Codec.Read(dsm.baseDir, s)
}

// WriteState records the entire intended state of the world to a dir.
func (dsm *DiskStateManager) WriteState(s *sous.State) error {
	return dsm.Codec.Write(dsm.baseDir, s)
}
