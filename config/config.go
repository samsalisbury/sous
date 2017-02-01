package config

import (
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path"

	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/util/firsterr"
	"github.com/pkg/errors"
)

type (
	// Config contains the core Sous configuration, shared by both the client and
	// server. The client and server may additionally have their own configuration.
	Config struct {
		// StateLocation is either a file containing a pre-compiled state, or
		// a directory containing the state as a tree.
		StateLocation string `env:"SOUS_STATE_LOCATION"`
		// Server is the location of a Sous Server which this sous instance
		// considers the master. If this is not set, this node is considered
		// to be a master. This value must be in URL format.
		Server string `env:"SOUS_SERVER"`
		// SiblingURLs is a temporary measure for setting up a distributed cluster
		// of sous servers. Each server must be configured with accessible URLs for
		// all the servers in production, as named by cluster.
		// (someday this should be replaced with a gossip protocol)
		SiblingURLs map[string]string
		// BuildStateDir is a directory where information about builds
		// performed by this user on this machine are stored.
		BuildStateDir string `env:"SOUS_BUILD_STATE_DIR"`
		// Docker is the Docker configuration.
		Docker docker.Config
		// User identifies the user of this client.
		User User
	}
	// User represents a user of the Sous client.
	User struct {
		// Name is the full name of this user.
		Name,
		// Email is the email address of this user.
		Email string
	}
)

func checkURL(URL, called string, args ...interface{}) error {
	called = fmt.Sprintf(called, args...)
	u, err := url.Parse(URL)
	if err != nil {
		return errors.Wrapf(err, "%s %q is not a valid URL", called, URL)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.Errorf("%s %q must begin with http:// or https://", called, URL)
	}
	return nil
}

// Validate returns an error if this config is invalid.
func (c Config) Validate() error {
	if c.Server != "" {
		if err := checkURL(c.Server, "Config.Server"); err != nil {
			return err
		}
	}
	for n, url := range c.SiblingURLs {
		if err := checkURL(url, "Config.SiblingURLs[%d]", n); err != nil {
			return err
		}
	}
	return nil
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Docker: docker.DefaultConfig(),
	}
}

// Equal compares
func (c *Config) Equal(other *Config) bool {
	if c.StateLocation != other.StateLocation {
		return false
	}
	if c.Server != other.Server {
		return false
	}
	if c.BuildStateDir != other.BuildStateDir {
		return false
	}
	if c.Docker != other.Docker {
		return false
	}
	if len(c.SiblingURLs) != len(other.SiblingURLs) {
		return false
	}
	for n, sib := range c.SiblingURLs {
		if other.SiblingURLs[n] != sib {
			return false
		}
	}
	return true
}

// FillDefaults fills in default values in this Config where they are currently
// zero values.
func (c *Config) FillDefaults() error {
	return firsterr.Set(
		func(e *error) {
			if c.StateLocation == "" {
				c.StateLocation, *e = c.defaultStateLocation()
			}
		},
		func(e *error) {
			*e = EnsureDirExists(c.StateLocation)
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
