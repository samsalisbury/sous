package config

import (
	"fmt"
	"os"
	"os/user"
	"path"

	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/util/firsterr"
)

// Config contains the core Sous configuration, shared by both the client and
// server. The client and server may additionally have their own configuration.
type (
	Config struct {
		// StateLocation is either a file containing a pre-compiled state, or
		// a directory containing the state as a tree.
		StateLocation string `env:"SOUS_STATE_LOCATION"`
		// Server is the location of a Sous Server which this sous instance
		// considers the master. If this is not set, this node is considered
		// to be a master.
		Server string `env:"SOUS_SERVER"`
		// BuildStateDir is a directory where information about builds
		// performed by this user on this machine are stored.
		BuildStateDir string `env:"SOUS_BUILD_STATE_DIR"`
		// Docker is the Docker configuration.
		Docker docker.Config
	}
)

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Docker: docker.DefaultConfig(),
	}
}

// FillDefaults fills in default values in this Config where they are currently
// zero values.
func (c *Config) FillDefaults() error {
	return firsterr.Parallel().Set(
		func(e *error) {
			if c.StateLocation == "" {
				c.StateLocation, *e = c.defaultStateLocation()
			}
		},
	)
}

// defaultStateLocation returns the default state location.
func (*Config) defaultStateLocation() (string, error) {
	dataRoot := os.Getenv("XDG_DATA_HOME")
	if dataRoot == "" {
		u, err := user.Current()
		if err != nil {
			return "", err
		}
		dataRoot = path.Join(u.HomeDir, ".local", "share")
	}
	stateLocation := path.Join(dataRoot, "sous", "state")
	return stateLocation, nil
}

// EnsureDirExists creates the named directory if it does not exist.
func EnsureDirExists(dir string) error {
	s, err := os.Stat(dir)
	if err == nil {
		if s.IsDir() {
			return nil
		}
		return fmt.Errorf("%q exists and is not a directory", dir)
	}
	if os.IsNotExist(err) || os.IsPermission(err) {
		return os.MkdirAll(dir, 0777)
	}
	return err
}
