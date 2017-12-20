package storage

import (
	"context"
	"database/sql"
	"fmt"

	// it's a SQL db driver. This is how you do that.
	_ "github.com/lib/pq"
	sous "github.com/opentable/sous/lib"
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
	return &PostgresStateManager{db: db}, nil
}

func (c PostgresConfig) connStr() string {
	return fmt.Sprintf("dbname=%s user=%s password=%s host=%s port=%d", c.DBName, c.User, c.Password, c.Host, c.Port)
}

// ReadState implements sous.StateReader on PostgresStateManager
func (m PostgresStateManager) ReadState() (*sous.State, error) {
	context := context.TODO()
	tx, err := m.db.BeginTx(context, nil)
	if err != nil {
		return nil, err
	}
	defer func(tx *sql.Tx) {
		// ignoring error - since if the Tx is committed, we would expect an error on rollback
		tx.Rollback()
	}(tx)

	state := sous.NewState()

	if err := loadEnvDefs(context, state, tx); err != nil {
		return nil, err
	}
	if err := loadResourceDefs(context, state, tx); err != nil {
		return nil, err
	}
	if err := loadMetadataDefs(context, state, tx); err != nil {
		return nil, err
	}
	if err := loadClusters(context, state, tx); err != nil {
		return nil, err
	}
	if err := loadManifests(context, state, tx); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return state, nil
}

func loadEnvDefs(context context.Context, state *sous.State, tx *sql.Tx) error {
	if err := loadTable(context,
		"select name, desc, scope, type from env_var_defs;",
		func(r *sql.DB) error {
			d := sous.EnvDef{}
			if err := rows.Scan(&d.Name, &d.Desc, &d.Scope, &d.Type); err != nil {
				return err
			}
			state.Defs.EnvVars = append(state.Defs.EnvVars, d)
		}); err != nil {
		return nil, err
	}
}

func loadResourceDefs(context context.Context, state *sous.State, tx *sql.Tx) error {
	if err := loadTable(context,
		"select field_name, var_type, default_value from resource_fdefs;",
		func(r *sql.DB) error {
			d := sous.FieldDefinition{}
			if err := rows.Scan(&d.Name, &d.Type, &d.Default); err != nil {
				return err
			}
			state.Defs.Resources = append(state.Defs.Resources, d)
		}); err != nil {
		return nil, err
	}
}

func loadMetadataDefs(context context.Context, state *sous.State, tx *sql.Tx) error {
	if err := loadTable(context,
		"select field_name, var_type, default_value from metadata_fdefs;",
		func(r *sql.DB) error {
			d := sous.FieldDefinition{}
			if err := rows.Scan(&d.Name, &d.Type, &d.Default); err != nil {
				return err
			}
			state.Defs.Metadata = append(state.Defs.Metadata, d)
		}); err != nil {
		return nil, err
	}
}

func loadMetadataDefs(context context.Context, state *sous.State, tx *sql.Tx) error {
	clusters := make(map[int]*Cluster)
	if err := loadTable(context,
		`select
		clusters.cluster_id, clusters.name, clusters.kind, base_url,
		crdef_skip, crdef_connect_delay, crdef_timeout, crdef_connect_interval,
		crdef_proto, crdef_path, crdef_port_index, crdef_failure_statuses,
		crdef_url_timeout, crdef_interval, crdef_retries,
		qualities.name
		from
		clusters
		natural join cluster_qualities
		natural join qualities
		where qualities.kind = "advisory";
		`,
		func(r *sql.DB) error {
			var cid int
			c := &sous.Cluster{}
			q := sous.Quality{}
			if err := rows.Scan(
				&cid, &c.Name, &c.Kind, &c.BaseURL,
				&c.Startup.Skip, &c.Startup.ConnectDelay, &c.Startup.Timeout, &c.Startup.ConnectInterval,
				&c.Startup.CheckReadyProtocol, &c.Startup.CheckReadyURIPath, &c.Startup.CheckReadyPortIndex, &c.Startup.CheckReadyFailureStatuses,
				&c.Startup.CheckReadyURITimeout, &c.Startup.CheckReadyInterval, &c.Startup.CheckReadyRetries,
				&q.Name,
			); err != nil {
				return err
			}
			if newC, has := clusters[cid]; has {
				c = newC
			} else {
				clusters[cid] = c
			}
			c.AllowedAdvisories = append(c.AllowedAdvisories, q.Name)
		}); err != nil {
		return nil, err
	}
	for _, c := range clusters {
		state.Defs.Metadata = append(state.Defs.Clusters, c)
	}
}

func loadManifests(context context.Context, state *sous.State, tx *sql.Tx) error {
	if err := loadTable(context,
		"select field_name, var_type, default_value from metadata_fdefs;",
		func(r *sql.DB) error {
			d := sous.FieldDefinition{}
			if err := rows.Scan(&d.Name, &d.Type, &d.Default); err != nil {
				return err
			}
			state.Defs.Metadata = append(state.Defs.Metadata, d)
		}); err != nil {
		return nil, err
	}
}

func loadTable(ctx context.Contex, tx *sql.Tx, sql string, pack func(*sql.Rows) error) error {
	rows, err := tx.QueryContext(ctx, "select name, desc, scope, type from env_fdefs;")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		if err := pack(rows); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return
	}
	return nil
}

// WriteState implements StateWriter on PostgresStateManager
func (m PostgresStateManager) WriteState(state *sous.State) error {
}
