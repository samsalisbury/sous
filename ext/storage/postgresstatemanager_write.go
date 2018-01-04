package storage

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"strings"
	"text/template"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

// WriteState implements StateWriter on PostgresStateManager
func (m PostgresStateManager) WriteState(state *sous.State, user sous.User) error {
	context := context.TODO()
	tx, err := m.db.BeginTx(context, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: false})
	if err != nil {
		return err
	}
	defer func(tx *sql.Tx) {
		// ignoring error - since if the Tx is committed, we would expect an error on rollback
		tx.Rollback()
	}(tx)

	if err := storeManifests(context, m.log, state, tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func storeManifests(ctx context.Context, log logging.LogSink, state *sous.State, tx *sql.Tx) error {
	newDeps, err := state.Deployments()
	if err != nil {
		return err
	}

	currentState, err := loadState(ctx, log, tx)
	if err != nil {
		return err
	}
	currentDeps, err := currentState.Deployments()
	if err != nil {
		return err
	}

	diffs := currentDeps.Diff(newDeps).Collect()
	updates := sous.NewDeployments()
	deletes := sous.NewDeployments()
	alldeps := sous.NewDeployments()

	for _, diff := range diffs {
		switch diff.Kind() {
		default: //do nothing for Same
		case sous.AddedKind, sous.ModifiedKind:
			updates.Add(diff.Post.Deployment)
			alldeps.Add(diff.Post.Deployment)
		case sous.RemovedKind:
			deletes.Add(diff.Prior.Deployment)
			alldeps.Add(diff.Prior.Deployment)
		}
	}

	if err := execInsertDeployments(ctx, log, tx, alldeps, "components", "on conflict do nothing", func(fields *fields, dep *sous.Deployment) {
		fields.row(func(r rowdef) {
			r.fd("'%s'", "repo", dep.SourceID.Location.Repo)
			r.fd("'%s'", "dir", dep.SourceID.Location.Dir)
			r.fd("'%s'", "flavor", dep.Flavor)
			r.fd("'%s'", "kind", dep.Kind)
		})
	}); err != nil {
		return nil
	}

	if err := execInsertDeployments(ctx, log, tx, alldeps, "clusters", `on conflict {{.Candidates}} do update set {{.NonCandidates}} = {{.NSNonCandidates "excluded"}}`, func(fields *fields, dep *sous.Deployment) {
		c := dep.Cluster
		s := c.Startup
		fields.row(func(r rowdef) {
			r.cf("'%s'", "name", dep.ClusterName)
			r.fd("'%s'", "kind", c.Kind)
			r.fd("'%s'", "base_url", c.BaseURL)
			startupFields(r, "crdef", s)
		})
	}); err != nil {
		return nil
	}

	if err := execInsertDeployments(ctx, log, tx, updates, "deployments", "", func(fields *fields, dep *sous.Deployment) {
		s := dep.Startup
		fields.row(func(r rowdef) {
			compID(r, dep)
			clusterID(r, dep)
			r.fd("'%s'", "versionstring", dep.SourceID.Version.String())
			r.fd("%d", "num_instances", dep.NumInstances)
			r.fd("'%s'", "schedule_string", dep.Schedule)
			r.fd("'%s'", "lifecycle", "active")
			startupFields(r, "cr", s)
		})
	}); err != nil {
		return err
	}

	if err := execInsertDeployments(ctx, log, tx, deletes, "deployments", "", func(fields *fields, dep *sous.Deployment) {
		s := dep.Startup
		fields.row(func(r rowdef) {
			compID(r, dep)
			clusterID(r, dep)
			r.fd("'%s'", "versionstring", dep.SourceID.Version.String())
			r.fd("%d", "num_instances", dep.NumInstances)
			r.fd("'%s'", "schedule_string", dep.Schedule)
			r.fd("'%s'", "lifecycle", "decommisioned")
			startupFields(r, "cr", s)
		})
	}); err != nil {
		return err
	}

	if err := execInsertDeployments(ctx, log, tx, updates, "owners", "on conflict do nothing", func(fields *fields, dep *sous.Deployment) {
		for ownername := range dep.Owners {
			fields.row(func(r rowdef) {
				r.fd("'%s'", "email", ownername)
			})
		}
	}); err != nil {
		return err
	}

	if err := execInsertDeployments(ctx, log, tx, updates, "component_owners", "on conflict do nothing", func(fields *fields, dep *sous.Deployment) {
		for ownername := range dep.Owners {
			fields.row(func(row rowdef) {
				compID(row, dep)
				ownerID(row, ownername)
			})
		}
	}); err != nil {
		return err
	}

	if err := execInsertDeployments(ctx, log, tx, updates, "envs", "on conflict do nothing", func(fields *fields, dep *sous.Deployment) {
		for key, value := range dep.Env {
			fields.row(func(row rowdef) {
				depID(row, dep)
				row.fd("'%s'", "key", key)
				row.fd("'%s'", "value", value)
			})
		}
	}); err != nil {
		return err
	}

	if err := execInsertDeployments(ctx, log, tx, updates, "resources", "on conflict do nothing", func(fields *fields, dep *sous.Deployment) {
		for key, value := range dep.Resources {
			fields.row(func(row rowdef) {
				depID(row, dep)
				row.fd("'%s'", "resource_name", key)
				row.fd("'%s'", "resource_value", value)
			})
		}
	}); err != nil {
		return err
	}

	if err := execInsertDeployments(ctx, log, tx, updates, "metadatas", "on conflict do nothing", func(fields *fields, dep *sous.Deployment) {
		for key, value := range dep.Metadata {
			fields.row(func(row rowdef) {
				depID(row, dep)
				row.fd("'%s'", "name", key)
				row.fd("'%s'", "value", value)
			})
		}
	}); err != nil {
		return err
	}

	if err := execInsertDeployments(ctx, log, tx, updates, "volumes", "on conflict do nothing", func(fields *fields, dep *sous.Deployment) {
		for _, volume := range dep.Volumes {
			fields.row(func(row rowdef) {
				depID(row, dep)
				row.fd("'%s'", "host", volume.Host)
				row.fd("'%s'", "container", volume.Container)
				row.fd("'%s'", "mode", volume.Mode)
			})
		}
	}); err != nil {
		return err
	}

	return nil
}

func depID(row rowdef, dep *sous.Deployment) {
	sid := dep.SourceID
	row.fd(`(select max(deployment_id)
	from
		deployments
		join components using (component_id)
		join clusters using (cluster_id)
	where
	  lifecycle = 'active' and
	  repo = '%s' and dir = '%s' and flavor = '%s' and components.kind = '%s' and clusters.name = '%s')`,
		"deployment_id", sid.Location.Repo, sid.Location.Dir, dep.Flavor, dep.Kind, dep.ClusterName)
}

func compID(row rowdef, dep *sous.Deployment) {
	sid := dep.SourceID
	row.fd(`(select component_id from components
	  where repo = '%s' and dir = '%s' and flavor = '%s' and kind = '%s')`,
		"component_id", sid.Location.Repo, sid.Location.Dir, dep.Flavor, dep.Kind)
}

func clusterID(row rowdef, dep *sous.Deployment) {
	row.fd(`(select "cluster_id" from clusters where name = '%s')`, "cluster_id", dep.ClusterName)
}

func ownerID(row rowdef, ownername string) {
	row.fd("(select owner_id from owners where email = '%s')", "owner_id", ownername)
}

func startupFields(r rowdef, prefix string, s sous.Startup) {
	r.fd("%t", prefix+"_skip", s.SkipCheck)
	r.fd("'%s'", prefix+"_proto", s.CheckReadyProtocol)
	r.fd("'%s'", prefix+"_path", s.CheckReadyURIPath)
	r.fd("%d", prefix+"_connect_delay", s.ConnectDelay)
	r.fd("%d", prefix+"_timeout", s.Timeout)
	r.fd("%d", prefix+"_connect_interval", s.ConnectInterval)
	r.fd("%d", prefix+"_port_index", s.CheckReadyPortIndex)
	r.fd("%d", prefix+"_uri_timeout", s.CheckReadyURITimeout)
	r.fd("%d", prefix+"_interval", s.CheckReadyInterval)
	r.fd("%d", prefix+"_retries", s.CheckReadyRetries)
	r.fd("%s", prefix+"_failure_statuses", sqlArray(s.CheckReadyFailureStatuses))
}

type fields struct {
	colnames []string
	coldefs  map[string]*coldef
	rows     []row
}

func (f *fields) getcol(col, frmt string, cand bool) *coldef {
	if c, has := f.coldefs[col]; has {
		if col != c.name || frmt != c.fmt || cand != c.candidate {
			panic(fmt.Sprintf("Mismatched coldef: %#v != %q %q", c, col, frmt))
		}
		return c
	}
	c := &coldef{name: col, fmt: frmt, candidate: cand}
	f.coldefs[col] = c
	f.colnames = append(f.colnames, col)
	return c
}

func (f *fields) row(fn func(rowdef)) {
	row := row{}
	f.rows = append(f.rows, row)
	def := rowdef{row: &row, fields: f}
	fn(def)
}

func (f fields) potent() bool {
	return len(f.colnames) > 0
}

func (f fields) insertSQL(table, conflict string) string {
	vs := f.values()
	return fmt.Sprintf("insert into %s %s values %s %s", table, f.columns(), vs, f.conflictClause(conflict))
}

func (f fields) conflictClause(templ string) string {
	buf := &bytes.Buffer{}
	conflictTemplate := template.Must(template.New("conflict").Parse(templ))
	conflictTemplate.Execute(buf, f)
	return buf.String()
}

func (f fields) columns() string {
	return "(" + strings.Join(f.colnames, ",") + ")"
}

// Candidates returns the index candidate columns for this fields.
func (f fields) Candidates() string {
	return f.candidates()
}

func (f fields) candidates() string {
	colnames := []string{}
	for _, name := range f.colnames {
		if f.coldefs[name].candidate {
			colnames = append(colnames, name)
		}
	}
	return "(" + strings.Join(colnames, ",") + ")"
}

// NonCandidates returns noncandidate column names for this fields.
func (f fields) NonCandidates() string {
	return f.noncandidates()
}

// NSNonCandidates returns noncandidate columns namespaced with a table name.
func (f fields) NSNonCandidates(namespace string) string {
	colnames := []string{}
	for _, name := range f.colnames {
		if !f.coldefs[name].candidate {
			colnames = append(colnames, namespace+"."+name)
		}
	}
	return "(" + strings.Join(colnames, ",") + ")"
}

func (f fields) noncandidates() string {
	colnames := []string{}
	for _, name := range f.colnames {
		if !f.coldefs[name].candidate {
			colnames = append(colnames, name)
		}
	}
	return "(" + strings.Join(colnames, ",") + ")"
}

func (f fields) values() string {
	valpats := []string{}
	for _, name := range f.colnames {
		valpats = append(valpats, f.coldefs[name].fmt)
	}
	format := "(" + strings.Join(valpats, ",") + ")"

	lines := []string{}
	for _, r := range f.rows {
		vals := []interface{}{}
		for _, name := range f.colnames {
			vals = append(vals, r[name].values...)
		}
		lines = append(lines, fmt.Sprintf(format, vals...))
	}
	return strings.Join(lines, ",\n")
}

func (f fields) rowcount() int {
	return len(f.rows)
}

type coldef struct {
	fmt, name string
	candidate bool
}

type row map[string]field

type rowdef struct {
	row    *row
	fields *fields
}

func (r rowdef) deffield(fmt string, col string, vals []interface{}, cand bool) {
	column := r.fields.getcol(col, fmt, cand)
	(*r.row)[col] = field{column: column, values: vals}
}

func (r rowdef) fd(fmt string, col string, vals ...interface{}) {
	r.deffield(fmt, col, vals, false)
}

func (r rowdef) cf(fmt string, col string, vals ...interface{}) {
	r.deffield(fmt, col, vals, true)
}

type field struct {
	column *coldef
	values []interface{}
}

func execInsertDeployments(
	ctx context.Context,
	log logging.LogSink,
	tx *sql.Tx,
	ds sous.Deployments,
	table string,
	conflict string,
	fn func(*fields, *sous.Deployment),
) error {
	fields := &fields{
		coldefs: map[string]*coldef{},
		rows:    []row{},
	}
	for _, d := range ds.Snapshot() {
		fn(fields, d)
	}
	if !fields.potent() {
		return nil
	}
	start := time.Now()
	sql := fields.insertSQL(table, conflict)
	_, err := tx.ExecContext(ctx, sql)
	reportSQLMessage(log, start, sql, fields.rowcount(), err)
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
	return "'{" + strings.Join(items, ",") + "}'"
}
