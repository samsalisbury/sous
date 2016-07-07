package docker

import (
	"database/sql"
	"fmt"

	// triggers the loading of sqlite3 as a database driver
	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	_ "github.com/mattn/go-sqlite3"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
)

type (
	// NameCache is a database for looking up SourceVersions based on
	// Docker image names and vice versa.
	NameCache struct {
		RegistryClient docker_registry.Client
		DB             *sql.DB
	}

	imageName string

	// NotModifiedErr is returned when an HTTP server returns Not Modified in
	// response to a conditional request
	NotModifiedErr struct{}

	// NoImageNameFound is returned when we cannot find an image name for a given
	// SourceVersion
	NoImageNameFound struct {
		sous.SourceVersion
	}

	// NoSourceVersionFound is returned when we cannot find a SourceVersion for a
	// given image name
	NoSourceVersionFound struct {
		imageName
	}
)

func DockerBuildArtifact(imageName string) *sous.BuildArtifact {
	return &sous.BuildArtifact{Name: imageName, Type: "docker"}
}

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
func NewNameCache(cl docker_registry.Client, db *sql.DB) *NameCache {
	return &NameCache{cl, db}
}

// GetArtifact implements sous.Registry.GetArtifact
func (nc *NameCache) GetArtifact(sv sous.SourceVersion) (*sous.BuildArtifact, error) {
	name, err := nc.getImageName(sv)
	if err != nil {
		return nil, err
	}
	return DockerBuildArtifact(name), nil
}

// GetSourceVersion looks up the source version for a given image name
func (nc *NameCache) GetSourceVersion(a *sous.BuildArtifact) (sous.SourceVersion, error) {
	in := a.Name
	var sv sous.SourceVersion

	Log.Vomit.Printf("Getting source version for %s", in)

	etag, repo, offset, version, _, err := nc.dbQueryOnName(in)
	if nif, ok := err.(NoSourceVersionFound); ok {
		Log.Vomit.Print(nif)
	} else if err != nil {
		Log.Vomit.Print("Err: ", err)
		return sous.SourceVersion{}, err
	} else {

		sv, err = makeSourceVersion(repo, offset, version)
		if err != nil {
			return sv, err
		}
	}

	md, err := nc.RegistryClient.GetImageMetadata(in, etag)
	Log.Vomit.Printf("%+ v %v %T %#v", md, err, err, err)
	if _, ok := err.(NotModifiedErr); ok {
		Log.Debug.Printf("Image name: %s -> Source version: %v", in, sv)
		return sv, nil
	}
	if err == distribution.ErrManifestNotModified {
		Log.Debug.Printf("Image name: %s -> Source version: %v", in, sv)
		return sv, nil
	}
	if err != nil {
		return sv, err
	}

	newSV, err := SourceVersionFromLabels(md.Labels)
	if err != nil {
		return sv, err
	}

	err = nc.dbInsert(newSV, md.Registry+"/"+md.CanonicalName, md.Etag)
	if err != nil {
		return sv, err
	}

	Log.Vomit.Printf("cn: %v all: %v", md.CanonicalName, md.AllNames)
	names := []string{}
	for _, n := range md.AllNames {
		names = append(names, md.Registry+"/"+n)
	}
	err = nc.dbAddNames(md.Registry+"/"+md.CanonicalName, names)

	Log.Debug.Printf("Image name: %s -> (updated) Source version: %v", in, newSV)
	return newSV, err
}

// GetImageName returns the docker image name for a given source version
func (nc *NameCache) getImageName(sv sous.SourceVersion) (string, error) {
	Log.Vomit.Printf("Getting image name for %+v", sv)
	cn, _, err := nc.dbQueryOnSV(sv)
	if _, ok := err.(NoImageNameFound); ok {
		err = nc.harvest(sv.CanonicalName())
		if err != nil {
			Log.Vomit.Printf("Err: %v", err)
			return "", err
		}

		cn, _, err = nc.dbQueryOnSV(sv)
		if err != nil {
			Log.Vomit.Printf("Err: %v", err)
			return "", err
		}
	} else if err != nil {
		return "", err
	}
	Log.Debug.Printf("Source version: %v -> image name %s", sv, cn)
	return cn, nil
}

// GetCanonicalName returns the canonical name for an image given any known name
func (nc *NameCache) GetCanonicalName(in string) (string, error) {
	_, _, _, _, cn, err := nc.dbQueryOnName(in)
	Log.Debug.Printf("Canonicalizing %s - got %s / %v", in, cn, err)
	return cn, err
}

// Insert puts a given SourceVersion/image name pair into the name cache
func (nc *NameCache) Insert(sv sous.SourceVersion, in, etag string) error {
	return nc.dbInsert(sv, in, etag)
}

func (nc *NameCache) harvest(sl sous.SourceLocation) error {
	repos, err := nc.dbQueryOnSL(sl)
	if err != nil {
		return err
	}
	for _, r := range repos {
		ref, err := reference.ParseNamed(r)
		if err != nil {
			return fmt.Errorf("%v for %v", err, r)
		}
		ts, err := nc.RegistryClient.AllTags(r)
		if err == nil {
			for _, t := range ts {
				Log.Debug.Printf("Harvested tag: %v", t)
				in, err := reference.WithTag(ref, t)
				if err == nil {
					a := DockerBuildArtifact(in.String())
					nc.GetSourceVersion(a) //pull it into the cache...
				}
			}
		}
	}
	return nil
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

type DBConfig struct {
	Driver, Connection string
}

func GetDatabase(cfg *DBConfig) (*sql.DB, error) {
	driver := "sqlite3"
	conn := InMemory
	if cfg != nil {
		if cfg.Driver != "" {
			driver = cfg.Driver
		}
		if cfg.Connection != "" {
			conn = cfg.Connection
		}
	}

	db, err := sql.Open(driver, conn) //only call once
	if err != nil {
		return nil, err
	}

	if err := sqlExec(db, "pragma foreign_keys = ON;"); err != nil {
		return nil, err
	}

	if err := sqlExec(db, "create table if not exists docker_repo_name("+
		"repo_name_id integer primary key autoincrement"+
		", name text not null"+
		", constraint upsertable unique (name) on conflict replace"+
		");"); err != nil {
		return nil, err
	}

	if err := sqlExec(db, "create table if not exists docker_search_location("+
		"location_id integer primary key autoincrement"+
		", repo text not null"+
		", offset text not null"+
		", constraint upsertable unique (repo, offset) on conflict replace"+
		");"); err != nil {
		return nil, err
	}

	if err := sqlExec(db, "create table if not exists repo_through_location("+
		"repo_name_id references docker_repo_name"+
		"   on delete cascade on update cascade not null"+
		", location_id references docker_search_location"+
		"   on delete cascade on update cascade not null"+
		",  primary key (repo_name_id, location_id) on conflict replace"+
		");"); err != nil {
		return nil, err
	}

	if err := sqlExec(db, "create table if not exists docker_search_metadata("+
		"metadata_id integer primary key autoincrement"+
		", location_id references docker_search_location"+
		"   on delete cascade on update cascade not null"+
		", etag text not null"+
		", canonicalName text not null"+
		", version text not null"+
		", constraint upsertable unique (location_id, version) on conflict replace"+
		", constraint canonical unique (canonicalName) on conflict replace"+
		");"); err != nil {
		return nil, err
	}

	if err := sqlExec(db, "create table if not exists docker_search_name("+
		"name_id integer primary key autoincrement"+
		", metadata_id references docker_search_metadata"+
		"   on delete cascade on update cascade not null"+
		", name text not null unique on conflict replace"+
		");"); err != nil {
		return nil, err
	}

	return db, err
}

func sqlExec(db *sql.DB, sql string) error {
	if _, err := db.Exec(sql); err != nil {
		return fmt.Errorf("Error: %s in SQL: %s", err, sql)
	}
	return nil
}

func (nc *NameCache) dbInsert(sv sous.SourceVersion, in, etag string) error {
	ref, err := reference.ParseNamed(in)
	Log.Debug.Printf("Parsed image name: %v", ref)
	if err != nil {
		return fmt.Errorf("%v for %v", err, in)
	}

	Log.Vomit.Printf("Inserting name %s", ref.Name())
	nr, err := nc.DB.Exec("insert into docker_repo_name "+
		"(name) values ($1);", ref.Name())
	nid, err := nr.LastInsertId()
	if err != nil {
		return err
	}

	res, err := nc.DB.Exec("insert into docker_search_location "+
		"(repo, offset) values ($1, $2);",
		string(sv.RepoURL), string(sv.RepoOffset))

	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	_, err = nc.DB.Exec("insert into repo_through_location "+
		"(repo_name_id, location_id) values ($1, $2)", nid, id)
	if err != nil {
		return err
	}

	Log.Vomit.Printf("Inserting metadata %v %v %v %v", id, etag, in, sv.Version)
	res, err = nc.DB.Exec("insert into docker_search_metadata "+
		"(location_id, etag, canonicalName, version) values ($1, $2, $3, $4);",
		id, etag, in, sv.Version.Format(semv.MMPPre))

	if err != nil {
		return err
	}

	id, err = res.LastInsertId()
	if err != nil {
		return err
	}

	res, err = nc.DB.Exec("insert into docker_search_name "+
		"(metadata_id, name) values ($1, $2)", id, in)

	return err
}

func (nc *NameCache) dbAddNames(cn string, ins []string) error {
	var id int
	Log.Debug.Printf("Adding names for %s: %+v", cn, ins)
	row := nc.DB.QueryRow("select metadata_id from docker_search_metadata "+
		"where canonicalName = $1", cn)
	err := row.Scan(&id)
	if err != nil {
		return err
	}
	add, err := nc.DB.Prepare("insert into docker_search_name " +
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
	row := nc.DB.QueryRow("select "+
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

func (nc *NameCache) dbQueryOnSL(sl sous.SourceLocation) (rs []string, err error) {
	rows, err := nc.DB.Query("select docker_repo_name.name "+
		"from "+
		"docker_repo_name natural join repo_through_location "+
		"  natural join docker_search_location "+
		"where "+
		"docker_search_location.repo = $1 and "+
		"docker_search_location.offset = $2",
		string(sl.RepoURL), string(sl.RepoOffset))

	if err == sql.ErrNoRows {
		return []string{}, err
	}
	if err != nil {
		return []string{}, err
	}

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

func (nc *NameCache) dbQueryOnSV(sv sous.SourceVersion) (cn string, ins []string, err error) {
	ins = make([]string, 0)
	rows, err := nc.DB.Query("select docker_search_metadata.canonicalName, "+
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

func makeSourceVersion(repo, offset, version string) (sous.SourceVersion, error) {
	v, err := semv.Parse(version)
	if err != nil {
		return sous.SourceVersion{}, err
	}

	return sous.SourceVersion{
		sous.RepoURL(repo), v, sous.RepoOffset(offset),
	}, nil
}
