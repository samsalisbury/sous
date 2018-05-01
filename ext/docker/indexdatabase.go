package docker

import (
	"context"
	"database/sql"

	"github.com/docker/distribution/reference"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/sqlgen"
)

func (nc *NameCache) dbInsert(sid sous.SourceID, in, etag string, quals []sous.Quality) error {
	ref, err := reference.ParseNamed(in)
	if err != nil {
		return err
	}

	ctx := context.TODO()
	tx, err := nc.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: false})
	if err != nil {
		return err
	}

	defer tx.Rollback() // we commit before returning...

	ins := sqlgen.NewInserter(ctx, nc.Log, tx)

	if err := ins.Exec("docker_repo_name", "", sqlgen.SingleRow(func(r sqlgen.RowDef) {
		r.KV("name", ref.Name())
	})); err != nil {
		return err
	}

	if err := ins.Exec("docker_search_location", "", sqlgen.SingleRow(func(r sqlgen.RowDef) {
		r.KV("repo", sid.Location.Repo)
		r.KV("offset", sid.Location.Dir)
	})); err != nil {
		return err
	}

	if err := ins.Exec("repo_through_location", "", sqlgen.SingleRow(func(r sqlgen.RowDef) {
		nameID(r, ref)
		locID(r, sid)
	})); err != nil {
		return err
	}

	if err := ins.Exec("docker_search_metadata", sqlgen.Upsert, sqlgen.SingleRow(func(r sqlgen.RowDef) {
		r.CF("?", "canonicalName", in)
		r.KV("etag", etag)
		r.KV("version", versionString)
		locID(r, sid)
	})); err != nil {
		return err
	}

	if err := ins.Exec("docker_image_qualities", sqlgen.DoNothing, func(fs sqlgen.FieldSet) {
		for _, q := range quals {
			if q.Kind == "advisory" && q.Name == "" {
				continue
			}
			fs.Row(func(r sqlgen.RowDef) {
				mdID(r, in)
				r.KV("quality", q.Name)
				r.KV("kind", q.Kind)
			})
		}
	}); err != nil {
		return err
	}

	if err := addSearchNames(ins, in, []string{in}); err != nil {
		return err
	}

	return tx.Commit()
}

func (nc *NameCache) dbAddNames(in string, names []string) error {
	ctx := context.TODO()
	tx, err := nc.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: false})
	if err != nil {
		return err
	}

	defer tx.Rollback() // we commit before returning...

	ins := sqlgen.NewInserter(ctx, nc.Log, tx)

	if err := addSearchNames(ins, in, names); err != nil {
		return err
	}

	return tx.Commit()
}

func addSearchNames(ins sqlgen.Inserter, in string, names []string) error {
	return ins.Exec("docker_search_name", "", func(fs sqlgen.FieldSet) {
		for _, n := range names {
			fs.Row(func(r sqlgen.RowDef) {
				mdID(r, in)
				r.KV("name", n)
			})
		}
	})
}

func nameID(r sqlgen.RowDef, ref reference.Named) {
	r.FD(`(select repo_name_id from docker_repo_name where name = ?)`, "repo_name_id", ref.Name())
}

func locID(r sqlgen.RowDef, sid sous.SourceID) {
	r.FD(`(select location_id from docker_search_location
	where repo = ? and offset = ?)`, "location_id", sid.Location.Repo, sid.Location.Dir)
}

func mdID(r sqlgen.RowDef, in string) {
	r.FD(`(select metadata_id from docker_search_metadata where canonicalName = ?)`, "metadata_id", in)
}
