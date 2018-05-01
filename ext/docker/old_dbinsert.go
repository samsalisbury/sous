package docker

import (
	"github.com/docker/distribution/reference"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/pkg/errors"
)

func (nc *NameCache) oldDBInsert(sid sous.SourceID, in, etag string, quals []sous.Quality) error {
	ref, err := reference.ParseNamed(in)
	messages.ReportLogFieldsMessage("Parsed image name from", logging.DebugLevel, nc.Log, ref, in)
	if err != nil {
		return errors.Errorf("%v for %v", err, in)
	}

	messages.ReportLogFieldsMessage("Inserting name for", logging.ExtraDebug1Level, nc.Log, ref.Name(), sid)

	var nid, id int64
	nid, err = nc.ensureInDB(
		"select repo_name_id from docker_repo_name where name = $1",
		"insert into docker_repo_name (name) values ($1);",
		ref.Name())

	if err != nil {
		return err
	}

	messages.ReportLogFieldsMessage("name -> id", logging.ExtraDebug1Level, nc.Log, ref.Name(), nid)

	id, err = nc.ensureInDB(
		"select location_id from docker_search_location where repo = $1 and offset = $2",
		"insert into docker_search_location (repo, offset) values ($1, $2);",
		sid.Location.Repo, sid.Location.Dir)

	if err != nil {
		return err
	}

	messages.ReportLogFieldsMessage("Source Loc -> id", logging.ExtraDebug1Level, nc.Log, sid.Location, id)

	_, err = nc.DB.Exec("insert or ignore into repo_through_location "+
		"(repo_name_id, location_id) values ($1, $2)", nid, id)
	if err != nil {
		return errors.Wrapf(err, "inserting (%d, %d) into repo_through_location", nid, id)
	}

	messages.ReportLogFieldsMessage("Inserting metadata id, etag, name, version", logging.ExtraDebug1Level, nc.Log, id, etag, in, versionString(sid.Version))

	id, err = nc.ensureInDB(
		"select metadata_id from docker_search_metadata  where canonicalName = $1",
		"insert or replace into docker_search_metadata (canonicalName, location_id, etag, version) values ($1, $2, $3, $4);",
		in, id, etag, versionString)

	if err != nil {
		return err
	}

	for _, q := range quals {
		if q.Kind == "advisory" && q.Name == "" {
			continue
		}
		nc.DB.Exec("insert into docker_image_qualities"+
			"  (metadata_id, quality, kind)"+
			"  values"+
			"  ($1,$2,$3)",
			id, q.Name, q.Kind)
	}

	messages.ReportLogFieldsMessage("Inserting search name", logging.ExtraDebug1Level, nc.Log, id, in)
	return nc.dbAddNamesForID(id, []string{in})
}

func (nc *NameCache) olddbAddNames(cn string, ins []string) error {
	var id int64
	messages.ReportLogFieldsMessage("Adding names for", logging.DebugLevel, nc.Log, cn, ins)
	row := nc.DB.QueryRow("select metadata_id from docker_search_metadata "+
		"where canonicalName = $1", cn)
	err := row.Scan(&id)
	if err != nil {
		return err
	}
	return nc.dbAddNamesForID(id, ins)
}

func (nc *NameCache) dbAddNamesForID(id int64, ins []string) error {
	add, err := nc.DB.Prepare("insert or replace into docker_search_name " +
		"(metadata_id, name) values ($1, $2)")
	if err != nil {
		return errors.Wrap(err, "adding names")
	}
	defer add.Close()

	for _, n := range ins {
		_, err := add.Exec(id, n)
		if err != nil {
			messages.ReportLogFieldsMessage("error dbAddNamesForID", logging.WarningLevel, nc.Log, errors.Cause(err), err)
			return errors.Wrapf(err, "adding name: %s", n)
		}
	}
	return nil
}
