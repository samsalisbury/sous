//+build smoke

package smoke

import "os"

type Instance struct {
	Addr                string
	StateDir, ConfigDir string
	ClusterName         string
	Proc                *os.Process
	LogDir              string
}
