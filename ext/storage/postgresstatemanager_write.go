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
		return err
	}
	defer func(tx *sql.Tx) {
		// ignoring error - since if the Tx is committed, we would expect an error on rollback
		tx.Rollback()
	}(tx)

	if err := storeManifests(context, state, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
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

	// TODO
	// Filter for changed deployments
	//

	// collect and insert owners
	// insert through records to owners

	// insert envs
	// insert resources
	// insert metadatas
	// insert volumes

	deps, err := state.Deployments()
	if err != nil {
		return err
	}

	if err := execInsert(ctx, tx, deps, "components", "on conflict do nothing", func(dep *sous.Deployment) fields {
		return fields{
			f("'%s'", "repo", il(dep.SourceID.Location.Repo)),
			f("'%s'", "dir", il(dep.SourceID.Location.Dir)),
			f("'%s'", "flavor", il(dep.Flavor)),
			f("'%s'", "kind", il(dep.Kind)),
		}
	}); err != nil {
		return nil
	}

	if err := execInsert(ctx, tx, deps, "clusters", "on conflict %s do update set %s = ROW", func(dep *sous.Deployment) fields {
		c := dep.Cluster
		s := c.Startup
		return append(fields{
			f("'%s'", "name", il(c.Name), true),
			f("'%s'", "kind", il(c.Kind)),
			f("'%s'", "base_url", il(c.BaseURL)),
		}, startupFields("crdef", s)...)
	}); err != nil {
		return nil
	}

	if err := execInsert(ctx, tx, deps, "deployments", "", func(dep *sous.Deployment) fields {
		sid := dep.SourceID
		s := dep.Startup
		return append(fields{
			f("(component_id from components where repo = '%s' and dir = '%s' and flavor = '%s' and kind = '%s')", "component_id", il(sid.Location.Repo, sid.Location.Dir, dep.Flavor, dep.Kind)),
			f("(select cluster_id from clusters where name = '%s')", "cluster_id", il(dep.ClusterName)),
			f("'%s'", "versionstring", il(dep.SourceID.Version.String())),
			f("%d", "num_instances", il(dep.NumInstances)),
			f("'%s'", "schedule_string", il(dep.Schedule)),
		}, startupFields("cr", s)...)
	}); err != nil {
		return err
	}

	return nil
}

func startupFields(prefix string, s sous.Startup) fields {
	return fields{
		f("%t", prefix+"_skip", il(s.SkipCheck)),
		f("'%s'", prefix+"_proto", il(s.CheckReadyProtocol)),
		f("'%s'", prefix+"_path", il(s.CheckReadyURIPath)),
		f("%d", prefix+"_connect_delay", il(s.ConnectDelay)),
		f("%d", prefix+"_timeout", il(s.Timeout)),
		f("%d", prefix+"_connect_interval", il(s.ConnectInterval)),
		f("%d", prefix+"_port_index", il(s.CheckReadyPortIndex)),
		f("%d", prefix+"_url_timeout", il(s.CheckReadyURITimeout)),
		f("%d", prefix+"_interval", il(s.CheckReadyInterval)),
		f("%d", prefix+"_retries", il(s.CheckReadyRetries)),
		f("%s", prefix+"_failure_statuses", il(sqlArray(s.CheckReadyFailureStatuses))),
	}
}

func il(vs ...interface{}) []interface{} {
	return vs
}

func f(fmt string, col string, vals []interface{}, cands ...bool) field {
	field := field{fmt: fmt, column: col, values: vals}
	if len(cands) > 0 {
		field.candidate = cands[0]
	}
	return field
}

type fields []field

type field struct {
	fmt, column string
	values      []interface{}
	candidate   bool
}

func execInsert(ctx context.Context, tx *sql.Tx, ds sous.Deployments, table string, conflict string, fields func(*sous.Deployment) fields) error {
	fs := fields(zeroDep())
	conflictClause := fmt.Sprintf(conflict, candidates(fs), noncandidates(fs))
	sql := fmt.Sprintf("insert into %s %s values %s %s", table, columns(fs), values(fs, ds, fields), conflictClause)
	_, err := tx.ExecContext(ctx, sql)
	return err
}

func zeroDep() *sous.Deployment {
	return &sous.Deployment{
		DeployConfig: sous.DeployConfig{
			Resources: map[string]string{},
			Metadata:  map[string]string{},
			Env:       map[string]string{},
			Volumes:   sous.Volumes{},
		},
		Cluster: &sous.Cluster{
			Env:               map[string]sous.Var{},
			Startup:           sous.Startup{},
			AllowedAdvisories: []string{},
		},
		Owners: map[string]struct{}{},
	}
}

func columns(fields fields) string {
	colnames := []string{}
	for _, field := range fields {
		colnames = append(colnames, field.column)
	}
	return "(" + strings.Join(colnames, ",") + ")"
}

func candidates(fields fields) string {
	colnames := []string{}
	for _, field := range fields {
		if field.candidate {
			colnames = append(colnames, field.column)
		}
	}
	return "(" + strings.Join(colnames, ",") + ")"
}

func noncandidates(fields fields) string {
	colnames := []string{}
	for _, field := range fields {
		if !field.candidate {
			colnames = append(colnames, field.column)
		}
	}
	return "(" + strings.Join(colnames, ",") + ")"
}

func values(fs fields, ds sous.Deployments, ff func(*sous.Deployment) fields) string {
	valpats := []string{}
	for _, field := range fs {
		valpats = append(valpats, field.fmt)
	}
	format := "(" + strings.Join(valpats, ",") + ")"

	lines := []string{}
	for _, d := range ds.Snapshot() {
		dfs := ff(d)
		vals := []interface{}{}
		for _, df := range dfs {
			vals = append(vals, df.values...)
		}
		lines = append(lines, fmt.Sprintf(format, vals...))
	}
	return strings.Join(lines, ",\n")
}

func sqlValues(ds sous.Deployments, format string, f func(*sous.Deployment) []interface{}) string {
	list := []string{}
	for _, d := range ds.Snapshot() {
		list = append(list, fmt.Sprintf(format, f(d)...))
	}
	return strings.Join(list, ",")
}

func sqlArray(value []int) string {
	items := []string{}
	for _, i := range value {
		items = append(items, fmt.Sprintf("%d", i))
	}
	return "{" + strings.Join(items, ",") + "}"
}
