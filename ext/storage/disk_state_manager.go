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
	"github.com/pkg/errors"
)

type (
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
	// TODO: Allow state dir to be passed as flag in sous/cli.
	// TODO: Consider returning a bool to indicate if the state dir exists at all.
	s := sous.NewState()
	err := dsm.Codec.Read(dsm.baseDir, s)
	if err != nil {
		return s, err
	}
	if s.Defs.Clusters == nil {
		return s, nil // errors.Errorf("no clusters defined in %s", dsm.baseDir)
	}
	for _, k := range s.Manifests.Keys() {
		m, _ := s.Manifests.Get(k)
		for clusterName := range m.Deployments {
			_, ok := s.Defs.Clusters[clusterName]
			if clusterName != "Global" && !ok {
				return s, errors.Errorf("cluster %q not defined (from manifest %q)",
					clusterName, k)
			}
		}
	}
	return s, nil
}

// WriteState records the entire intended state of the world to a dir.
func (dsm *DiskStateManager) WriteState(s *sous.State) error {
	return dsm.Codec.Write(dsm.baseDir, s)
}
