package storage

import (
	"database/sql"
	"fmt"

	// it's a SQL db driver. This is how you do that.
	_ "github.com/lib/pq"
)

type (
	// The PostgresStateManager provides the StateManager interface by
	// reading/writing from a postgres database.
	PostgresStateManager struct {
		db *sql.DB
	}

	// A PostgresConfig describes how to connect to a postgres database
	PostgresConfig struct {
		DBName   string
		User     string
		Password string
		Host     string
		Port     int
	}
)

// NewPostgresStateManager creates a new PostgresStateManager.
func NewPostgresStateManager(cfg PostgresConfig) (*PostgresStateManager, error) {
	db, err := sql.Open("postgres", cfg.connStr())
	if err != nil {
		return nil, err
	}
	return &PostgresStateManager{db: db}
}

func (c PostgresConfig) connStr() string {
	fmt.Sprintf("dbname=%s user=%s password=%s host=%s port=%d", c.DBName, c.User, c.Password, c.Host, c.Port)
}

// ReadState implements sous.StateReader on PostgresStateManager
func (m PostgresStateManager) ReadState() (*sous.State, error) {
	context := context.TODO()
	tx, err := DB.BeginTx(context, nil)
	if err != nil {
		return nil, err
	}
	defer func(tx *db.Tx) {
		// ignoring error - since if the Tx is committed, we would expect an error on rollback
		tx.Rollback()
	}(tx)

	state := NewState()

	rows, err := tx.QueryContext(context, "select field_name, var_type, default_value from env_fdefs;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		d := EnvDef{}
		EnvDef{}
		err := rows.Scan()

	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return state, nil
}

// WriteState implements StateWriter on PostgresStateManager
func (m PostgresStateManager) WriteState(state *sous.State) error {
}
