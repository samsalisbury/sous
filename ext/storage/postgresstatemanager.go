package storage

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/lib/pq" // need to load the postgres driver
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
		DBName   string `env:"SOUS_PG_DBNAME"`
		User     string `env:"SOUS_PG_USER"`
		Password string `env:"SOUS_PG_PASSWORD"`
		Host     string `env:"SOUS_PG_HOST"`
		Port     string `env:"SOUS_PG_PORT"`
		SSL      bool   `env:"SOUS_PG_SSL"`
	}
)

// NewPostgresStateManager creates a new PostgresStateManager.
func NewPostgresStateManager(db *sql.DB, log logging.LogSink) *PostgresStateManager {
	return &PostgresStateManager{db: db, log: log}
}

func (c PostgresConfig) connStr() string {
	conn := []string{}
	if c.Host != "" {
		conn = append(conn, fmt.Sprintf("host=%s", c.Host))
	}
	if c.Port != "" {
		conn = append(conn, fmt.Sprintf("port=%s", c.Port))
	}
	if !c.SSL {
		conn = append(conn, "sslmode=disable")
	} else {
		conn = append(conn, "sslmode=enable")
	}

	if c.DBName != "" {
		conn = append(conn, fmt.Sprintf("dbname=%s", c.DBName))
	}

	if c.User != "" {
		conn = append(conn, fmt.Sprintf("user=%s", c.User))
	}
	if c.Password != "" {
		conn = append(conn, fmt.Sprintf("password=%s", c.Password))
	}
	return strings.Join(conn, " ")
}

// DB returns a database connection based on this config
func (c PostgresConfig) DB() (*sql.DB, error) {
	db, err := sql.Open("postgres", c.connStr())
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		return nil, err
	}
	return db, nil
}
