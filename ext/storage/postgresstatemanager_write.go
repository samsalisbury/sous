package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	sous "github.com/opentable/sous/lib"
)

// WriteState implements StateWriter on PostgresStateManager
func (m PostgresStateManager) WriteState(state *sous.State) error {
	context := context.TODO()
	tx, err := m.db.BeginTx(context, nil)
	if err != nil {
		return nil, err
	}
	defer func(tx *sql.Tx) {
		// ignoring error - since if the Tx is committed, we would expect an error on rollback
		tx.Rollback()
	}(tx)

	if err := storeManifests(context, state, tx); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return state, nil
}

func storeManifests(ctx context.Context, state *sous.State, tx *sql.Tx) error {
	/*
		`select
			repo, dir, flavor, kind,
			email,
			versionstring, num_instances, schedule_string,
			cr_skip, cr_connect_delay, cr_timeout, cr_connect_interval,
			cr_proto, cr_path, cr_port_index, cr_failure_statuses,
			cr_url_timeout, cr_interval, cr_retries,
			clusters.name,
			envs.key, envs.value,
			resource_name, resource_value,
			metadatas.name, metadatas.value,
			host, container, mode
		from
			components
			join owner_components using component_id
			join owners using owner_id
			join deployments using component_id
			join clusters using cluster_id
			left join envs using deployment_id
			left join resources using deployment_id
			left join metadata using deployment_id
			left join volumes using deployment_id
		where deployment_id in (
			select max(deployment_id) from deployments group by cluster_id, component_id
		)
		`,
	*/

	deps, err := state.Deployments()
	if err != nil {
		return err
	}

	asVals := sqlValues(deps, "(%q, %q, %q, %q)", func(dep *sous.Deployment) []interface{} {
		return []interface{}{
			dep.SourceID.Location.Repo,
			dep.SourceID.Location.Dir,
			dep.Flavor,
			dep.Kind,
		}
	})

	tx.ExecContext(ctx, fmt.Sprintf(
		`insert into
		components (repo, dir, flavor, kind) values %s
		on conflict do nothing
		returning component_id, repo, dir, flavor, kind`, values))
}

func sqlValues(ds sous.Deployments, format string, f func(*sous.Deployment) []interface{}) string {
	list := []string{}
	for _, d := range ds.Snapshot() {
		list = append(list, fmt.Sprintf(format, f(d)...))
	}
	return strings.Join(list, ",")
}
