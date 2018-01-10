package storage

import (
	"database/sql"
	"fmt"

	// it's a SQL db driver. This is how you do that.
	_ "github.com/lib/pq"
	"github.com/opentable/sous/util/logging"
)

type (
	// The PostgresStateManager provides the StateManager interface by
	// reading/writing from a postgres database.
	PostgresStateManager struct {
		db  *sql.DB
		log logging.LogSink
	}

	// A PostgresConfig describes how to connect to a postgres database
	PostgresConfig struct {
		DBName   string
		User     string
		Password string
		Host     string
		Port     string
		SSL      bool
	}
)

// NewPostgresStateManager creates a new PostgresStateManager.
func NewPostgresStateManager(db *sql.DB, log logging.LogSink) *PostgresStateManager {
	return &PostgresStateManager{db: db, log: log}
}

func (c PostgresConfig) connStr() string {
	sslmode := "enable"
	if !c.SSL {
		sslmode = "disable"
	}
	conn := fmt.Sprintf("dbname=%s host=%s port=%s sslmode=%s", c.DBName, c.Host, c.Port, sslmode)
	if c.User != "" {
		conn = fmt.Sprintf("%s user=%s", conn, c.User)
	}
	if c.Password != "" {
		conn = fmt.Sprintf("%s password=%s", conn, c.Password)
	}
	return conn
}

// DB returns a database connection based on this config
func (c PostgresConfig) DB() (*sql.DB, error) {
	return sql.Open("postgres", c.connStr())
}
