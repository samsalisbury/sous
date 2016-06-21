package sous

import (
	"database/sql"
	"fmt"
	"log"

	// triggers the loading of sqlite3 as a database driver
	_ "github.com/mattn/go-sqlite3"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
)

type (
	// NameCache is a database for looking up SourceVersions based on
	// Docker image names and vice versa.
	NameCache struct {
		registryClient docker_registry.Client
		db             *sql.DB
	}

	imageName string

	// NotModifiedErr is returned when an HTTP server returns Not Modified in
	// response to a conditional request
	NotModifiedErr struct{}

	// NoImageNameFound is returned when we cannot find an image name for a given
	// SourceVersion
	NoImageNameFound struct {
		SourceVersion
	}

	// NoSourceVersionFound is returned when we cannot find a SourceVersion for a
	// given image name
	NoSourceVersionFound struct {
		imageName
	}

	// ImageMapper interface describes the component responsible for mapping
	// source versions to names
	ImageMapper interface {
		// GetCanonicalName returns the canonical name for an image given any known
		// name
		GetCanonicalName(in string) (string, error)

		// Insert puts a given SourceVersion/image name pair into the name cache
		Insert(sv SourceVersion, in, etag string) error

		// GetImageName returns the docker image name for a given source version
		GetImageName(sv SourceVersion) (string, error)

		// GetSourceVersion returns the source version for a given image name
		GetSourceVersion(in string) (SourceVersion, error)
	}
)

// InMemory configures SQLite to use an in-memory database
// The dummy file allows multiple goroutines see the same in-memory DB
const InMemory = "file:dummy.db?mode=memory&cache=shared"

// InMemoryConnection builds a connection string based on a base name
// This is mostly useful for testing, so that we can have separate cache DBs per test
func InMemoryConnection(base string) string {
	return "file:" + base + "?mode=memory&cache=shared"
}

func (e NoImageNameFound) Error() string {
	return fmt.Sprintf("No image name for %v", e.SourceVersion)
}

func (e NoSourceVersionFound) Error() string {
	return fmt.Sprintf("No source version for %v", e.imageName)
}

func (e NotModifiedErr) Error() string {
	return "Not modified"
}

// NewNameCache builds a new name cache
func NewNameCache(cl docker_registry.Client, dbCfg ...string) *NameCache {
	db, err := getDatabase(dbCfg...)
	if err != nil {
		log.Fatal("Error building name cache DB: ", err)
	}

	return &NameCache{cl, db}
}

// GetSourceVersion looks up the source version for a given image name
func (nc *NameCache) GetSourceVersion(in string) (SourceVersion, error) {
	var sv SourceVersion

	Log.Debug.Print(in)

	etag, repo, offset, version, _, err := nc.dbQueryOnName(in)
	Log.Debug.Print(repo, offset, version, err)
	if nif, ok := err.(NoSourceVersionFound); ok {
		Log.Debug.Print(nif)
	} else if err != nil {
		Log.Debug.Print(err)
		return SourceVersion{}, err
	} else {
		Log.Debug.Print(repo, offset, version)

		sv, err = makeSourceVersion(repo, offset, version)
		if err != nil {
			return sv, err
		}
	}

	md, err := nc.registryClient.GetImageMetadata(in, etag)
	Log.Debug.Print(md, err)
	if _, ok := err.(NotModifiedErr); ok {
		return sv, nil
	}
	if err != nil {
		return sv, err
	}

	newSV, err := SourceVersionFromLabels(md.Labels)
	if err != nil {
		return sv, err
	}

	nc.dbInsert(newSV, md.CanonicalName, md.Etag)
	nc.dbAddNames(md.CanonicalName, md.AllNames)

	return newSV, nil
}

// GetCanonicalName returns the canonical name for an image given any known name
func (nc *NameCache) GetCanonicalName(in string) (string, error) {
	_, _, _, _, cn, err := nc.dbQueryOnName(in)
	return cn, err
}

// Insert puts a given SourceVersion/image name pair into the name cache
func (nc *NameCache) Insert(sv SourceVersion, in, etag string) error {
	return nc.dbInsert(sv, in, etag)
}

// GetImageName returns the docker image name for a given source version
func (nc *NameCache) GetImageName(sv SourceVersion) (string, error) {
	cn, _, err := nc.dbQueryOnSV(sv)
	if err != nil {
		return "", err
	}
	return cn, nil
}

func union(left, right []string) []string {
	set := make(map[string]struct{})
	for _, s := range left {
		set[s] = struct{}{}
	}

	for _, s := range right {
		set[s] = struct{}{}
	}

	res := make([]string, 0, len(set))

	for k := range set {
		res = append(res, k)
	}

	return res
}

func getDatabase(cfg ...string) (*sql.DB, error) {
	driver := "sqlite3"
	conn := InMemory
	if len(cfg) >= 1 {
		driver = cfg[0]
	}

	if len(cfg) >= 2 {
		conn = cfg[1]
	}

	db, err := sql.Open(driver, conn) //only call once
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("create table if not exists docker_search_location(" +
		"location_id integer primary key autoincrement, " +
		"repo text not null, " +
		"offset text not null," +
		"constraint upsertable unique (repo, offset) on conflict replace" +
		");")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("create table if not exists docker_search_metadata(" +
		"metadata_id integer primary key autoincrement, " +
		"location_id references docker_search_location " +
		"   on delete cascade on update cascade not null, " +
		"etag text not null, " +
		"canonicalName text not null, " +
		"version text not null, " +
		"constraint upsertable unique (location_id, version) on conflict replace" +
		");")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("create table if not exists docker_search_name(" +
		"name_id integer primary key autoincrement, " +
		"metadata_id references docker_search_metadata " +
		"   on delete cascade on update cascade not null, " +
		"name text not null unique on conflict replace" +
		");")

	_, err = db.Exec("pragma foreign_keys = ON;")
	if err != nil {
		return nil, err
	}

	return db, err
}

func (nc *NameCache) dbInsert(sv SourceVersion, in, etag string) error {
	res, err := nc.db.Exec("insert into docker_search_location "+
		"(repo, offset) values ($1, $2);",
		string(sv.RepoURL), string(sv.RepoOffset))

	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	res, err = nc.db.Exec("insert into docker_search_metadata "+
		"(location_id, etag, canonicalName, version) values ($1, $2, $3, $4);",
		id, etag, in, sv.Version.Format(semv.MMPPre))

	if err != nil {
		return err
	}

	id, err = res.LastInsertId()
	if err != nil {
		return err
	}

	res, err = nc.db.Exec("insert into docker_search_name "+
		"(metadata_id, name) values ($1, $2)", id, in)

	return err
}

func (nc *NameCache) dbAddNames(cn string, ins []string) error {
	var id int
	row := nc.db.QueryRow("select metadata_id from docker_search_metadata "+
		"where canonicalName = $1", cn)
	err := row.Scan(&id)
	if err != nil {
		return err
	}
	add, err := nc.db.Prepare("insert into docker_search_name " +
		"(metadata_id, name) values ($1, $2)")
	if err != nil {
		return err
	}

	for _, n := range ins {
		_, err := add.Exec(id, n)
		if err != nil {
			return err
		}
	}

	return nil
}

func (nc *NameCache) dbQueryOnName(in string) (etag, repo, offset, version, cname string, err error) {
	row := nc.db.QueryRow("select "+
		"docker_search_metadata.etag, "+
		"docker_search_location.repo, "+
		"docker_search_location.offset, "+
		"docker_search_metadata.version, "+
		"docker_search_metadata.canonicalName "+
		"from "+
		"docker_search_name natural join docker_search_metadata "+
		"natural join docker_search_location "+
		"where docker_search_name.name = $1", in)
	err = row.Scan(&etag, &repo, &offset, &version, &cname)
	if err == sql.ErrNoRows {
		err = NoSourceVersionFound{imageName(in)}
	}
	return
}

func (nc *NameCache) dbQueryOnSV(sv SourceVersion) (cn string, ins []string, err error) {
	ins = make([]string, 0)
	rows, err := nc.db.Query("select docker_search_metadata.canonicalName, "+
		"docker_search_name.name "+
		"from "+
		"docker_search_name natural join docker_search_metadata "+
		"natural join docker_search_location "+
		"where "+
		"docker_search_location.repo = $1 and "+
		"docker_search_location.offset = $2 and "+
		"docker_search_metadata.version = $3",
		string(sv.RepoURL), string(sv.RepoOffset), sv.Version.String())

	if err == sql.ErrNoRows {
		err = NoImageNameFound{sv}
		return
	}
	if err != nil {
		return
	}

	for rows.Next() {
		var in string
		rows.Scan(&cn, &in)
		ins = append(ins, in)
	}
	err = rows.Err()
	if len(ins) == 0 {
		err = NoImageNameFound{sv}
	}

	return
}

func makeSourceVersion(repo, offset, version string) (SourceVersion, error) {
	v, err := semv.Parse(version)
	if err != nil {
		return SourceVersion{}, err
	}

	return SourceVersion{
		RepoURL(repo), v, RepoOffset(offset),
	}, nil
}
