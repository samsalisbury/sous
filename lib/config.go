package sous

import (
	"fmt"
	"os"
	"os/user"
	"path"

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
		// BuildStateLocation is a directory where information about builds
		// performed by this user on this machine are stored.
		BuildStateDir string `env:"SOUS_BUILD_STATE_DIR"`
		// DatabaseDriver is the name of the driver to use for local persistence
		DatabaseDriver string `env:"SOUS_DB_DRIVER"`
		// DatabaseConnection is the database connection string for local persistence
		DatabaseConnection string `env:"SOUS_DB_CONN"`
	}
)

// InMemory configures SQLite to use an in-memory database
// The dummy file allows multiple goroutines see the same in-memory DB
const InMemory = "file:dummy.db?mode=memory&cache=shared"

// DefaultConfig builds a default configuration, which can be then overridden by
// client code.
func DefaultConfig() Config {
	return Config{
		DatabaseDriver:     "sqlite3",
		DatabaseConnection: InMemory,
	}
}

func (c *Config) FillDefaults() error {
	return firsterr.Parallel().Set(
		func(e *error) { c.StateLocation, *e = c.defaultStateLocation() },
	)
}

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
	s, err := os.Stat(stateLocation)
	if err == nil {
		if s.IsDir() {
			return stateLocation, nil
		}
		return "", fmt.Errorf("state location %q exists and is not a directory",
			stateLocation)
	}
	if os.IsNotExist(err) || os.IsPermission(err) {
		if err := os.MkdirAll(stateLocation, 0777); err != nil {
			return "", fmt.Errorf("unable to create state location dir: %s", err)
		}
		return stateLocation, nil
	}
	return "", fmt.Errorf("unable to get state location: %s", err)
}
