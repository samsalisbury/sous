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
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/hy"
)

// ReadState loads the state of the world from a dir
func ReadState(dir string) (*sous.State, error) {
	s := &sous.State{}
	return s, hy.Unmarshal(dir, s)
}

// WriteState records the state of the world to a dir
func WriteState(dir string, s *sous.State) error {
	return hy.Marshal(dir, s)
}
