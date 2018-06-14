package docker

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/docker/distribution/reference"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/sqlgen"
	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
)

func (nc *NameCache) captureRepos(db *sql.DB) (repos []string) {
	res, err := db.Query("select name from docker_repo_name;")
	if err != nil {
		logging.WarnMsg(nc.log, "captureRepos", err)
		return
	}
	defer res.Close()
	for res.Next() {
		var repo string
		res.Scan(&repo)
		repos = append(repos, repo)
	}
	return
}

func versionString(v semv.Version) string {
	return v.Format(semv.MMPPre)
}

func revisionString(v semv.Version) string {
	// Lop off the + symbol, not needed when storing revision as a separate field.
	return v.Format("+")[1:]
}

func canonicalizeDockerReference(in string) (reference.Canonical, error) {
	ref, err := reference.ParseNamed(in)
	if err != nil {
		return nil, errors.Wrapf(err, "name: %q", in)
	}
	cn, hasDigest := ref.(reference.Canonical)
	if !hasDigest {
		return nil, errors.Errorf("docker reference lacks digest: %v", ref)
	}
	if _, hasTag := cn.(reference.Tagged); hasTag {
		digest := cn.Digest()
		if ref, err = reference.ParseNamed(ref.Name()); err != nil {
			return nil, err
		}

		if cn, err = reference.WithDigest(ref, digest); err != nil {
			return nil, err
		}
	}
	return cn, nil

}

func (nc *NameCache) dbInsert(sid sous.SourceID, in, etag string, extraNames []string, quals []sous.Quality) error {
	ref, err := canonicalizeDockerReference(in)
	if err != nil {
		return err
	}
	in = ref.String()

	ctx := context.TODO()
	tx, err := nc.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: false})
	if err != nil {
		return err
	}

	defer tx.Rollback() // we commit before returning...

	ins := sqlgen.NewInserter(ctx, nc.log, tx)

	if err := ins.Exec("docker_repo_name", sqlgen.DoNothing, sqlgen.SingleRow(func(r sqlgen.RowDef) {
		r.KV("name", ref.Name())
	})); err != nil {
		return err
	}

	if err := ins.Exec("docker_search_location", sqlgen.DoNothing, sqlgen.SingleRow(func(r sqlgen.RowDef) {
		r.KV("repo", sid.Location.Repo)
		r.KV("offset", sid.Location.Dir)
	})); err != nil {
		return err
	}

	if err := ins.Exec("repo_through_location", sqlgen.DoNothing, sqlgen.SingleRow(func(r sqlgen.RowDef) {
		nameID(r, ref)
		locID(r, sid)
	})); err != nil {
		return err
	}

	if err := ins.Exec("docker_search_metadata", sqlgen.Upsert, sqlgen.SingleRow(func(r sqlgen.RowDef) {
		r.CF("?", "canonicalname", in)
		r.KV("version", versionString(sid.Version))
		r.KV("revision", revisionString(sid.Version))
		r.KV("etag", etag)
		locID(r, sid)
	})); err != nil {
		// other errors should also get wrapped
		return errors.Wrapf(err, "canonicalname:%q version:%q etag:%q repo:%q dir:%q", in, sid.Version, etag, sid.Location.Repo, sid.Location.Dir)
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

	if err := addSearchNames(ins, in, append(extraNames, in)); err != nil {
		return err
	}

	nc.dumpTx(os.Stderr, tx)
	return tx.Commit()
}

func (nc *NameCache) dbAddNames(in string, names []string) error {
	ref, err := canonicalizeDockerReference(in)
	if err != nil {
		return err
	}
	in = ref.String()

	ctx := context.TODO()
	tx, err := nc.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: false})
	if err != nil {
		return err
	}

	defer tx.Rollback() // we commit before returning...

	ins := sqlgen.NewInserter(ctx, nc.log, tx)

	if err := addSearchNames(ins, in, names); err != nil {
		return err
	}

	return tx.Commit()
}

// n.B. this can only use `in` without canonicalization because all its callers have canonicalized it already.
func addSearchNames(ins sqlgen.Inserter, in string, names []string) error {
	return ins.Exec("docker_search_name", sqlgen.DoNothing, func(fs sqlgen.FieldSet) {
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
	where "repo" = ? and "offset" = ?)`, "location_id", sid.Location.Repo, sid.Location.Dir)
}

func candLocID(r sqlgen.RowDef, sid sous.SourceID) {
	r.CF(`(select location_id from docker_search_location
	where "repo" = ? and "offset" = ?)`, "location_id", sid.Location.Repo, sid.Location.Dir)
}

func mdID(r sqlgen.RowDef, in string) {
	r.FD(`(select metadata_id from docker_search_metadata where canonicalname = ?)`, "metadata_id", in)
}

// XXX

func (nc *NameCache) dbQueryOnName(in string) (etag, repo, offset, version, revision, cname string, err error) {
	row := nc.DB.QueryRow("select "+
		"docker_search_metadata.etag, "+
		"docker_search_location.repo, "+
		"docker_search_location.offset, "+
		"docker_search_metadata.version, "+
		"docker_search_metadata.revision, "+
		"docker_search_metadata.canonicalname "+
		"from "+
		"docker_search_name natural join docker_search_metadata "+
		"natural join docker_search_location "+
		"where docker_search_name.name = $1", in)
	err = row.Scan(&etag, &repo, &offset, &version, &revision, &cname)
	if err == sql.ErrNoRows {
		err = NoSourceIDFound{imageName(in)}
	}
	return
}

func (nc *NameCache) dbQueryOnSL(sl sous.SourceLocation) (rs []string, err error) {
	rows, err := nc.DB.Query("select docker_repo_name.name "+
		"from "+
		"docker_repo_name natural join repo_through_location "+
		"  natural join docker_search_location "+
		"where "+
		"docker_search_location.repo = $1 and "+
		"docker_search_location.offset = $2",
		string(sl.Repo), string(sl.Dir))

	if err == sql.ErrNoRows {
		return []string{}, err
	}
	if err != nil {
		return []string{}, err
	}
	defer rows.Close()

	for rows.Next() {
		var r string
		rows.Scan(&r)
		rs = append(rs, r)
	}
	err = rows.Err()
	if len(rs) == 0 {
		err = fmt.Errorf("no repos found for %+v", sl)
	}
	return
}

func (nc *NameCache) dbQueryAllSourceIds() (ids []sous.SourceID, err error) {
	rows, err := nc.DB.Query("select docker_search_location.repo, " +
		"docker_search_location.offset, " +
		"docker_search_metadata.version, " +
		"docker_search_metadata.revision " +
		"from " +
		"docker_search_location natural join docker_search_metadata")
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var repo, offset, version, revision string
		rows.Scan(&repo, &offset, &version, &revision)
		ids = append(ids, sous.SourceID{
			Location: sous.SourceLocation{
				Repo: repo, Dir: offset,
			},
			Version: semv.MustParse(version + "+" + revision),
		})
	}
	err = rows.Err()
	return
}

type strpairs []strpair
type strpair [2]string

func (nc *NameCache) dbQueryOnSourceID(sid sous.SourceID) (cn string, ins []string, quals strpairs, err error) {
	cn, ins, err = nc.dbQueryCNameforSourceID(sid)
	if err != nil {
		return
	}
	quals, err = nc.dbQueryQualsForCName(cn)
	return
}

func (nc *NameCache) dbQueryCNameforSourceID(sid sous.SourceID) (cn string, ins []string, err error) {
	start := time.Now()
	query := "select docker_search_metadata.canonicalname, " +
		"docker_search_name.name " +
		"from " +
		"docker_search_name natural join docker_search_metadata " +
		"natural join docker_search_location " +
		"where " +
		"docker_search_location.repo = $1 and " +
		"docker_search_location.offset = $2 and " +
		"docker_search_metadata.version = $3"
	rows, err := nc.DB.Query(query, sid.Location.Repo, sid.Location.Dir, versionString(sid.Version))

	if err != nil {
		sqlgen.ReportSelect(nc.log, start, "docker_search_metadata", query, 0, err,
			sid.Location.Repo, sid.Location.Dir, versionString(sid.Version))
		if err == sql.ErrNoRows {
			err = errors.Wrap(NoImageNameFound{sid}, "")
		}
		return
	}
	defer rows.Close()

	rowcount := 0
	for rows.Next() {
		rowcount++
		var in string
		rows.Scan(&cn, &in)
		ins = append(ins, in)
	}
	err = rows.Err()
	sqlgen.ReportSelect(nc.log, start, "docker_search_metadata", query, rowcount, err,
		sid.Location.Repo, sid.Location.Dir, versionString(sid.Version))
	if len(ins) == 0 {
		err = errors.Wrap(NoImageNameFound{sid}, "")
	}

	return
}

func (nc *NameCache) dbQueryQualsForCName(cn string) (quals strpairs, err error) {
	rows, err := nc.DB.Query("select"+
		" docker_image_qualities.quality,"+
		" docker_image_qualities.kind"+
		"   from"+
		" docker_image_qualities natural join docker_search_metadata"+
		" where"+
		" docker_search_metadata.canonicalname = $1", cn)

	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var pr strpair
		rows.Scan(&pr[0], &pr[1])
		quals = append(quals, pr)
	}
	err = rows.Err()

	return

}
