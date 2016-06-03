package sous

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

// DefaultConfig builds a default configuation, which can be then overridden by
// client code
func DefaultConfig() Config {
	return Config{
		DatabaseDriver:     "sqlite3",
		DatabaseConnection: ":memory:",
	}
}
