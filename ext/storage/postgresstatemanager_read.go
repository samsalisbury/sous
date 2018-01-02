package storage

import (
	"context"
	"database/sql"

	sous "github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
)

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

	if err := loadEnvDefs(context, tx, state); err != nil {
		return nil, err
	}
	if err := loadResourceDefs(context, tx, state); err != nil {
		return nil, err
	}
	if err := loadMetadataDefs(context, tx, state); err != nil {
		return nil, err
	}
	if err := loadClusters(context, tx, state); err != nil {
		return nil, err
	}
	if err := loadManifests(context, tx, state); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return state, nil
}

func loadEnvDefs(context context.Context, tx *sql.Tx, state *sous.State) error {
	if err := loadTable(context, tx,
		`select "name", "desc", "scope", "type" from env_var_defs;`,
		func(rows *sql.Rows) error {
			d := sous.EnvDef{}
			if err := rows.Scan(&d.Name, &d.Desc, &d.Scope, &d.Type); err != nil {
				return err
			}
			state.Defs.EnvVars = append(state.Defs.EnvVars, d)
			return nil
		}); err != nil {
		return err
	}
	return nil
}

func loadResourceDefs(context context.Context, tx *sql.Tx, state *sous.State) error {
	if err := loadTable(context, tx,
		`select "field_name", "var_type", "default_value" from resource_fdefs;`,
		func(rows *sql.Rows) error {
			d := sous.FieldDefinition{}
			if err := rows.Scan(&d.Name, &d.Type, &d.Default); err != nil {
				return err
			}
			state.Defs.Resources = append(state.Defs.Resources, d)
			return nil
		}); err != nil {
		return err
	}
	return nil
}

func loadMetadataDefs(context context.Context, tx *sql.Tx, state *sous.State) error {
	if err := loadTable(context, tx,
		`select "field_name", "var_type", "default_value" from metadata_fdefs;`,
		func(rows *sql.Rows) error {
			d := sous.FieldDefinition{}
			if err := rows.Scan(&d.Name, &d.Type, &d.Default); err != nil {
				return err
			}
			state.Defs.Metadata = append(state.Defs.Metadata, d)
			return nil
		}); err != nil {
		return err
	}
	return nil
}

func loadClusters(context context.Context, tx *sql.Tx, state *sous.State) error {
	clusters := make(map[int]*sous.Cluster)
	if err := loadTable(context, tx,
		`select
		clusters.cluster_id, clusters.name, clusters.kind, "base_url",
		"crdef_skip", "crdef_connect_delay", "crdef_timeout", "crdef_connect_interval",
		"crdef_proto", "crdef_path", "crdef_port_index", "crdef_failure_statuses",
		"crdef_url_timeout", "crdef_interval", "crdef_retries",
		qualities.name
		from
			clusters
			join cluster_qualities using (cluster_id)
			join qualities using (quality_id)
		where qualities.kind = 'advisory';
		`,
		func(rows *sql.Rows) error {
			var cid int
			c := &sous.Cluster{}
			q := sous.Quality{}
			if err := rows.Scan(
				&cid, &c.Name, &c.Kind, &c.BaseURL,
				&c.Startup.SkipCheck, &c.Startup.ConnectDelay, &c.Startup.Timeout, &c.Startup.ConnectInterval,
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
			return nil
		}); err != nil {
		return err
	}
	for _, c := range clusters {
		state.Defs.Clusters[c.Name] = c
	}
	return nil
}

func loadManifests(context context.Context, tx *sql.Tx, state *sous.State) error {
	if err := loadTable(context, tx,
		// This query is somewhat naive and returns many more rows than we need
		// specifically, every possible combination of env/resource/volume/metadata
		// results in its own row. Maybe that could be reduced?
		`select
			"repo", "dir", "flavor", clusters.kind,
			"email",
			"versionstring", "num_instances", "schedule_string",
			"cr_skip", "cr_connect_delay", "cr_timeout", "cr_connect_interval",
			"cr_proto", "cr_path", "cr_port_index", "cr_failure_statuses",
			"cr_uri_timeout", "cr_interval", "cr_retries",
			clusters.name,
			envs.key, envs.value,
			"resource_name", "resource_value",
			metadatas.name, metadatas.value,
			"host", "container", "mode"
		from
			components
			join component_owners using (component_id)
			join owners using (owner_id)
			join deployments using (component_id)
			join clusters using (cluster_id)
			left join envs using (deployment_id)
			left join resources using (deployment_id)
			left join metadatas using (deployment_id)
			left join volumes using (deployment_id)
		where deployment_id in (
			select max(deployment_id) from deployments group by cluster_id, component_id
		)
		and deployments.lifecycle != 'decommisioned'
		`,
		func(rows *sql.Rows) error {
			m := &sous.Manifest{}
			ds := sous.DeploySpec{}
			vol := sous.Volume{}
			var ownerEmail, versionString,
				clusterName,
				envKey, envValue,
				resName, resValue,
				mdName, mdValue string
			if err := rows.Scan(
				&m.Source.Repo, &m.Source.Dir, &m.Flavor, &m.Kind,
				&ownerEmail,
				&versionString, &ds.NumInstances, &ds.Schedule,
				&ds.Startup.SkipCheck, &ds.Startup.ConnectDelay, &ds.Startup.Timeout, &ds.Startup.ConnectInterval,
				&ds.Startup.CheckReadyProtocol, &ds.Startup.CheckReadyURIPath, &ds.Startup.CheckReadyPortIndex, &ds.Startup.CheckReadyFailureStatuses,
				&ds.Startup.CheckReadyURITimeout, &ds.Startup.CheckReadyInterval, &ds.Startup.CheckReadyRetries,
				&clusterName,
				&envKey, &envValue,
				&resName, &resValue,
				&mdName, &mdValue,
				&vol.Host, &vol.Container, &vol.Mode,
			); err != nil {
				return err
			}
			if newM, has := state.Manifests.Get(m.ID()); has {
				m = newM
			} else {
				state.Manifests.Add(m)
			}
			set := sous.NewOwnerSet(m.Owners...)
			set.Add(ownerEmail)
			m.Owners = set.Slice()
			if newDS, has := m.Deployments[clusterName]; has {
				ds = newDS
			} else {
				var err error
				if ds.Version, err = semv.Parse(versionString); err != nil {
					return err
				}
			}
			ds.Env[envKey] = envValue
			ds.Resources[resName] = resValue
			ds.Metadata[mdName] = mdValue
			has := false
			for i := range ds.Volumes {
				if ds.Volumes[i].Equal(&vol) {
					has = true
					break
				}
			}
			if !has {
				ds.Volumes = append(ds.Volumes, &vol)
			}
			m.Deployments[clusterName] = ds
			return nil
		}); err != nil {
		return err
	}
	return nil
}

func loadTable(ctx context.Context, tx *sql.Tx, sql string, pack func(*sql.Rows) error) error {
	rows, err := tx.QueryContext(ctx, sql)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		if err := pack(rows); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return nil
}
