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
	// NameCache is a primative database for looking up SourceVersions based on Docker image names and vice versa.
	NameCache struct {
		registryClient docker_registry.Client
		db             *sql.DB
		dockerNameLookup
		sourceNameLookup
	}

	imageName string

	sourceNameLookup map[imageName]*sourceRecord
	dockerNameLookup map[SourceVersion]*sourceRecord

	// NotModifiedErr is returned when an HTTP server returns Not Modified in
	// response to a conditional request
	NotModifiedErr struct{}

	// NoImageNameFound is returned when we cannot find an image name for a given SourceVersion
	NoImageNameFound struct {
		SourceVersion
	}

	// NoSourceVersionFound is returned when we cannot find a SourceVersion for a given image name
	NoSourceVersionFound struct {
		imageName
	}

	sourceRecord struct {
		md docker_registry.Metadata
	}
)

func (e NoImageNameFound) Error() string {
	return fmt.Sprintf("No image name for %v", e.SourceVersion)
}

func (e NoSourceVersionFound) Error() string {
	return fmt.Sprintf("No image name for %v", e.imageName)
}

func (e NotModifiedErr) Error() string {
	return "Not modified"
}

var theNameCache = NewNameCache(docker_registry.NewClient())

func getDatabase() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", ":memory:") //only call once
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("pragma foreign_keys = ON;")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("create table if not exists metadata(" +
		"metadata_id integer primary key autoincrement, " +
		"etag text not null, " +
		"canonicalName text not null, " +
		"repo text not null, " +
		"offset text not null," +
		"version text not null, " +
		"unique (repo, offset, version) on conflict replace" +
		");")
	if err != nil {
		return nil, err
	}

	_, err = db.Exec("create table if not exists name(" +
		"name_id integer primary key autoincrement, " +
		"metadata_id references metadata on delete cascade on update cascade not null, " +
		"name text not null unique on conflict replace" +
		");")

	return db, err
}

// NewNameCache builds a new name cache
func NewNameCache(cl docker_registry.Client) NameCache {
	db, err := getDatabase()
	if err != nil {
		log.Fatal(err)
	}

	return NameCache{
		cl,
		db,
		make(dockerNameLookup),
		make(sourceNameLookup),
	}
}

func (sr *sourceRecord) SourceVersion() (SourceVersion, error) {
	return SourceVersionFromLabels(sr.md.Labels)
}

func (sr *sourceRecord) Update(other *sourceRecord) {
	sr.md = other.md
}

// InsertImageRecord stores a SourceVersion/image name pair into the global name cache
func InsertImageRecord(sv SourceVersion, in, etag string) error {
	return theNameCache.Insert(sv, in, etag)
}

// GetImageName looks up the image name for a given SourceVersion - uses the global name cache
func GetImageName(sv SourceVersion) (string, error) {
	return theNameCache.GetImageName(sv)
}

// GetSourceVersion retreives a source version for an image name, updating it from the server if necessary
// Each call to GetSourceVersion implies an HTTP request, although it may be abbreviated by the use of an etag.
func GetSourceVersion(in string) (SourceVersion, error) {
	return theNameCache.GetSourceVersion(in)
}

// Insert puts a given SourceVersion/image name pair into the name cache
func (nc *NameCache) Insert(sv SourceVersion, in, etag string) error {
	nc.dbInsert(sv, in, etag)

	record := sourceRecord{docker_registry.Metadata{
		CanonicalName: in,
		AllNames:      []string{in},
		Etag:          etag,
		Labels:        sv.DockerLabels(),
	}}

	return nc.insertRecord(&record)
}

func (dl dockerNameLookup) GetImageName(sv SourceVersion) (string, error) {
	if sr, ok := dl[sv]; ok {
		return sr.md.CanonicalName, nil
	}
	return "", NoImageNameFound{sv}
}

func (nc *NameCache) dbInsert(sv SourceVersion, in, etag string) error {
	res, err := nc.db.Exec("insert into metadata (etag, canonicalName, repo, offset, version) values ($1, $2, $3, $4, $5);",
		etag, in, sv.RepoURL, sv.RepoOffset, sv.Version.String())

	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	_, err = nc.db.Exec("insert into name (metadata_id, name) values ($1, $2)", id, in)

	return err
}

func (nc *NameCache) dbAddNames(cn string, ins []string) error {
	var id int
	row := nc.db.QueryRow("select metadata_id from metadata where canonicalName = $1", cn)
	err := row.Scan(&id)
	if err != nil {
		return err
	}
	add, err := nc.db.Prepare("insert into name (metadata_id, name) values ($1, $2)")
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
	row := nc.db.QueryRow("select metadata.etag, metadata.repo, metadata.offset, metadata.version, metadata.canonicalName from name natural join metadata where name.name = $1", in)
	err = row.Scan(&etag, &repo, &offset, &version, &cname)
	return
}

func (nc *NameCache) dbQueryOnSV(sv SourceVersion) (cn string, ins []string, err error) {
	ins = make([]string, 0)
	rows, err := nc.db.Query("select metadata.canonicalName, name.name from name natural join metadata where "+
		"metadata.repo = $1 and metadata.offset = $2 and metadata.version = $3",
		sv.RepoURL, sv.RepoOffset, sv.Version.String())
	if err != nil {
		return
	}

	for rows.Next() {
		var in string
		rows.Scan(&cn, &in)
		ins = append(ins, in)
	}
	err = rows.Err()

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

// GetSourceVersion looks up the source version for a given image name
func (nc *NameCache) GetSourceVersion(in string) (SourceVersion, error) {
	etag, repo, offset, version, _, err := nc.dbQueryOnName(in)
	if err != nil {
		return SourceVersion{}, err
	}

	sv, err := makeSourceVersion(repo, offset, version)
	if err != nil {
		return sv, err
	}

	md, err := nc.registryClient.GetImageMetadata(in, etag)
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

func (sn sourceNameLookup) GetCanonicalName(in string) (string, error) {

	if sr, ok := sn[imageName(in)]; ok {
		return sr.md.CanonicalName, nil
	}
	return "", NoSourceVersionFound{imageName(in)}
}

func (nc *NameCache) insertRecord(sr *sourceRecord) error {
	err := nc.insertSourceVersion(sr)
	if err != nil {
		return err
	}

	return nc.insertDockerName(sr)
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

func (sn sourceNameLookup) getSourceVersion(in string) (SourceVersion, error) {
	if sr, ok := sn[imageName(in)]; ok {
		return sr.SourceVersion()
	}
	return SourceVersion{}, NoSourceVersionFound{imageName(in)}
}

func (sn sourceNameLookup) getSourceRecord(in imageName) (*sourceRecord, error) {
	if sr, ok := sn[in]; ok {
		return sr, nil
	}
	return nil, NoSourceVersionFound{in}
}

func (sn sourceNameLookup) insertSourceVersion(sr *sourceRecord) error {
	for _, n := range sr.md.AllNames {
		existing, yes := sn[imageName(n)]
		if yes {
			existing.Update(sr)
		} else {
			sn[imageName(n)] = sr
		}
	}
	return nil
}

func (dl dockerNameLookup) insertDockerName(sr *sourceRecord) error {
	sv, err := sr.SourceVersion()
	if err != nil {
		return err
	}

	existing, yes := dl[sv]
	if yes {
		existing.Update(sr)
	} else {
		dl[sv] = sr
	}
	return nil
}
