// The storage package is responsible for the persistent storage of state.
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

func ReadState(dir string) (*sous.State, error) {
	s := &sous.State{}
	return s, hy.Unmarshal(dir, s)
}

func WriteState(dir string, s *sous.State) error {
	return hy.Marshal(dir, s)
}
