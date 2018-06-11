package storage

import (
	"context"
	"database/sql"
	"time"

	"github.com/lib/pq"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/sqlgen"
	"github.com/pkg/errors"
)

// WriteState implements StateWriter on PostgresStateManager
func (m PostgresStateManager) WriteState(state *sous.State, user sous.User) error {
	start := time.Now()
	context := context.TODO()
	tx, err := m.db.BeginTx(context, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: false})
	if err != nil {
		reportWriting(m.log, start, state, errors.Wrapf(err, "opening transaction"))
		return err
	}
	defer func(tx *sql.Tx) {
		// ignoring error - since if the Tx is committed, we would expect an error on rollback
		tx.Rollback()
	}(tx)

	if err := storeManifests(context, m.log, state, tx); err != nil {
		reportWriting(m.log, start, state, errors.Wrapf(err, "storing state"))
		return err
	}

	if err := tx.Commit(); err != nil {
		reportWriting(m.log, start, state, errors.Wrapf(err, "committing transaction"))
		return err
	}
	reportWriting(m.log, start, state, nil)
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

	/* XXX consider logging this
	currentDeps.Len(),
	newDeps.Len(),
	updates.Len(),
	deletes.Len(),
	alldeps.Len(),
	*/

	ins := sqlgen.NewInserter(ctx, log, tx)

	if err := ins.Exec("components", sqlgen.DoNothing,
		deploymentsFieldSetter(alldeps, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			fields.Row(func(r sqlgen.RowDef) {
				r.FD("?", "repo", dep.SourceID.Location.Repo)
				r.FD("?", "dir", dep.SourceID.Location.Dir)
				r.FD("?", "flavor", dep.Flavor)
				r.FD("?", "kind", dep.Kind)
			})
		})); err != nil {
		return err
	}

	if err := ins.Exec("clusters", sqlgen.Upsert,
		deploymentsFieldSetter(alldeps, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			c := dep.Cluster
			s := c.Startup
			fields.Row(func(r sqlgen.RowDef) {
				r.CF("?", "name", dep.ClusterName)
				r.FD("?", "kind", c.Kind)
				r.FD("?", "base_url", c.BaseURL)
				startupFields(r, "crdef", s)
			})
		})); err != nil {
		return err
	}

	if err := ins.Exec("qualities", sqlgen.DoNothing,
		deploymentsFieldSetter(alldeps, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			c := dep.Cluster
			for _, adv := range c.AllowedAdvisories {
				fields.Row(func(r sqlgen.RowDef) {
					r.FD("?", "name", adv)
					r.FD("?", "kind", "advisory")
				})
			}
		})); err != nil {
		return err
	}

	// XXX As written, this makes it hard to delete advisories.
	if err := ins.Exec("cluster_qualities", sqlgen.DoNothing,
		deploymentsFieldSetter(alldeps, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			c := dep.Cluster
			for _, adv := range c.AllowedAdvisories {
				fields.Row(func(r sqlgen.RowDef) {
					clusterID(r, dep)
					advisoryID(r, adv)
				})
			}
		})); err != nil {
		return err
	}

	// We use application diffs for deployments (instead of upserts) because
	// otherwise it would be impossible to return to a previous state for a
	// manifest. Since rollback is a concrete use case, we do not want e.g.
	// "ON CONFLICT DO NOTHING", since there would be a previous identical state.
	if err := ins.Exec("deployments", "",
		deploymentsFieldSetter(updates, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			s := dep.Startup
			fields.Row(func(r sqlgen.RowDef) {
				compID(r, dep)
				clusterID(r, dep)
				r.FD("?", "versionstring", dep.SourceID.Version.String())
				r.FD("?", "num_instances", dep.NumInstances)
				r.FD("?", "schedule_string", dep.Schedule)
				r.FD("?", "lifecycle", "active")
				startupFields(r, "cr", s)
			})
		})); err != nil {
		return err
	}

	// see above - this is the conterpart insert for "deletes", which we're
	// tombstoning here.
	if err := ins.Exec("deployments", "",
		deploymentsFieldSetter(deletes, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			s := dep.Startup
			fields.Row(func(r sqlgen.RowDef) {
				compID(r, dep)
				clusterID(r, dep)
				r.FD("?", "versionstring", dep.SourceID.Version.String())
				r.FD("?", "num_instances", dep.NumInstances)
				r.FD("?", "schedule_string", dep.Schedule)
				r.FD("?", "lifecycle", "decommisioned")
				startupFields(r, "cr", s)
			})
		})); err != nil {
		return err
	}

	if err := ins.Exec("owners", sqlgen.DoNothing,
		deploymentsFieldSetter(updates, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			for ownername := range dep.Owners {
				fields.Row(func(r sqlgen.RowDef) {
					r.FD("?", "email", ownername)
				})
			}
		})); err != nil {
		return err
	}

	if err := ins.Exec("component_owners", sqlgen.DoNothing,
		deploymentsFieldSetter(updates, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			for ownername := range dep.Owners {
				fields.Row(func(row sqlgen.RowDef) {
					compID(row, dep)
					ownerID(row, ownername)
				})
			}
		})); err != nil {
		return err
	}

	if err := ins.Exec("envs", sqlgen.DoNothing,
		deploymentsFieldSetter(updates, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			for key, value := range dep.Env {
				fields.Row(func(row sqlgen.RowDef) {
					depID(row, dep)
					row.FD("?", "key", key)
					row.FD("?", "value", value)
				})
			}
		})); err != nil {
		return err
	}

	if err := ins.Exec("resources", sqlgen.DoNothing,
		deploymentsFieldSetter(updates, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			for key, value := range dep.Resources {
				fields.Row(func(row sqlgen.RowDef) {
					depID(row, dep)
					row.FD("?", "resource_name", key)
					row.FD("?", "resource_value", value)
				})
			}
		})); err != nil {
		return err
	}

	if err := ins.Exec("metadatas", sqlgen.DoNothing,
		deploymentsFieldSetter(updates, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			for key, value := range dep.Metadata {
				fields.Row(func(row sqlgen.RowDef) {
					depID(row, dep)
					row.FD("?", "name", key)
					row.FD("?", "value", value)
				})
			}
		})); err != nil {
		return err
	}

	if err := ins.Exec("volumes", sqlgen.DoNothing,
		deploymentsFieldSetter(updates, func(fields sqlgen.FieldSet, dep *sous.Deployment) {
			for _, volume := range dep.Volumes {
				fields.Row(func(row sqlgen.RowDef) {
					depID(row, dep)
					row.FD("?", "host", volume.Host)
					row.FD("?", "container", volume.Container)
					row.FD("?", "mode", volume.Mode)
				})
			}
		})); err != nil {
		return err
	}

	return nil
}

func depID(row sqlgen.RowDef, dep *sous.Deployment) {
	sid := dep.SourceID
	row.FD(`(select max(deployment_id)
	from
		deployments
		join components using (component_id)
		join clusters using (cluster_id)
	where
	  lifecycle = 'active' and
	  repo = ? and dir = ? and flavor = ? and components.kind = ? and clusters.name = ?)`,
		"deployment_id", sid.Location.Repo, sid.Location.Dir, dep.Flavor, dep.Kind, dep.ClusterName)
}

func compID(row sqlgen.RowDef, dep *sous.Deployment) {
	sid := dep.SourceID
	row.FD(`(select component_id from components
	  where repo = ? and dir = ? and flavor = ? and kind = ?)`,
		"component_id", sid.Location.Repo, sid.Location.Dir, dep.Flavor, dep.Kind)
}

func clusterID(row sqlgen.RowDef, dep *sous.Deployment) {
	row.FD(`(select "cluster_id" from clusters where name = ?)`, "cluster_id", dep.ClusterName)
}

func advisoryID(row sqlgen.RowDef, advName string) {
	row.FD(`(select "quality_id" from qualities where name = ?)`, "quality_id", advName)
}

func ownerID(row sqlgen.RowDef, ownername string) {
	row.FD("(select owner_id from owners where email = ?)", "owner_id", ownername)
}

func startupFields(r sqlgen.RowDef, prefix string, s sous.Startup) {
	statuses := []int64{}
	for _, n := range s.CheckReadyFailureStatuses {
		statuses = append(statuses, int64(n))
	}
	r.FD("?", prefix+"_skip", s.SkipCheck)
	r.FD("?", prefix+"_proto", s.CheckReadyProtocol)
	r.FD("?", prefix+"_path", s.CheckReadyURIPath)
	r.FD("?", prefix+"_connect_delay", s.ConnectDelay)
	r.FD("?", prefix+"_timeout", s.Timeout)
	r.FD("?", prefix+"_connect_interval", s.ConnectInterval)
	r.FD("?", prefix+"_port_index", s.CheckReadyPortIndex)
	r.FD("?", prefix+"_uri_timeout", s.CheckReadyURITimeout)
	r.FD("?", prefix+"_interval", s.CheckReadyInterval)
	r.FD("?", prefix+"_retries", s.CheckReadyRetries)
	r.FD("?", prefix+"_failure_statuses", pq.Array(statuses))
}

func deploymentsFieldSetter(ds sous.Deployments, eachDep func(sqlgen.FieldSet, *sous.Deployment)) func(sqlgen.FieldSet) {
	return func(fields sqlgen.FieldSet) {
		for _, d := range ds.Snapshot() {
			eachDep(fields, d)
		}
	}
}
