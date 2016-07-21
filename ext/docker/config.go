package docker

type Config struct {
	// DatabaseDriver is the name of the driver to use for local
	// persistence.
	DatabaseDriver string `env:"SOUS_DB_DRIVER"`
	// DatabaseConnection is the database connection string for local
	// persistence.
	DatabaseConnection string `env:"SOUS_DB_CONN"`
}

// DefaultConfig builds a default configuration, which can be then overridden by
// client code.
func DefaultConfig() Config {
	return Config{
		DatabaseDriver:     "sqlite3",
		DatabaseConnection: InMemory,
	}
}

func (c Config) DBConfig() DBConfig {
	return DBConfig{Driver: c.DatabaseDriver,
		Connection: c.DatabaseConnection,
	}
}
