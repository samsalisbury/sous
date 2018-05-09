package docker

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/docker/distribution"
	"github.com/docker/distribution/reference"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
)

type (
	// NameCache is a database for looking up SourceIDs based on
	// Docker image names and vice versa.
	NameCache struct {
		sync.Mutex
		RegistryClient     docker_registry.Client
		DB                 *sql.DB
		DockerRegistryHost string
		Log                logging.LogSink
		groomOnce          sync.Once
	}

	imageName string

	// NotModifiedErr is returned when an HTTP server returns Not Modified in
	// response to a conditional request
	NotModifiedErr struct{}

	// NoImageNameFound is returned when we cannot find an image name for a
	// given SourceID.
	NoImageNameFound struct {
		sous.SourceID
	}

	// NoSourceIDFound is returned when we cannot find a SourceID for a
	// given image name
	NoSourceIDFound struct {
		imageName
	}
)

// NewBuildArtifact creates a new sous.BuildArtifact representing a Docker
// image.
// XXX this should be removed in favor of sous.NewBuildArtifact
func NewBuildArtifact(imageName string, qstrs strpairs) *sous.BuildArtifact {
	var qs []sous.Quality
	for _, qstr := range qstrs {
		qs = append(qs, sous.Quality{Name: qstr[0], Kind: qstr[1]})
	}

	return &sous.BuildArtifact{Name: imageName, Type: "docker", Qualities: qs}
}

func (e NoImageNameFound) Error() string {
	return fmt.Sprintf("No image name for %v", e.SourceID)
}

func (e NoSourceIDFound) Error() string {
	return fmt.Sprintf("No source ID for %v", e.imageName)
}

func (e NotModifiedErr) Error() string {
	return "Not modified"
}

// NewNameCache builds a new name cache.
// XXX remove error return value
func NewNameCache(drh string, cl docker_registry.Client, ls logging.LogSink, db *sql.DB) (*NameCache, error) {
	nc := &NameCache{
		RegistryClient:     cl,
		DB:                 db,
		DockerRegistryHost: drh,
		Log:                ls,
	}
	return nc, nil
}

// ListSourceIDs lists all the known SourceIDs.
func (nc *NameCache) ListSourceIDs() ([]sous.SourceID, error) {
	return nc.dbQueryAllSourceIds()
}

// Warmup warms up the cache.
func (nc *NameCache) Warmup(r string) error {
	ref, err := reference.ParseNamed(r)
	if err != nil {
		return errors.Errorf("%v for %v", err, r)
	}
	ts, err := nc.RegistryClient.AllTags(r)
	if err != nil {
		return errors.Wrapf(err, "warming up %q", r)
	}
	for _, t := range ts {
		messages.ReportLogFieldsMessage("Harvested Tag", logging.DebugLevel, nc.Log, t, r)
		in, err := reference.WithTag(ref, t)
		if err == nil {
			a := NewBuildArtifact(in.String(), strpairs{})
			nc.GetSourceID(a) //pull it into the cache...
		} else {
			messages.ReportLogFieldsMessage("t loop", logging.WarningLevel, nc.Log, in, err)
		}
	}
	return nil
}

func (nc *NameCache) warmupSingle(sid sous.SourceID) error {
	in := versionTag(nc.DockerRegistryHost, sid, "")

	a := NewBuildArtifact(in, strpairs{})
	gsid, err := nc.GetSourceID(a)

	if err != nil {
		return nil
	}

	if !sid.Equal(gsid) {
		return errors.Errorf("Fetched %q for image name %q, was looking for %q", gsid, in, sid)
	}

	return nil

}

// ImageLabels gets the labels for an image name.
func (nc *NameCache) ImageLabels(in string) (map[string]string, error) {
	a := NewBuildArtifact(in, nil)
	sv, err := nc.GetSourceID(a)
	if err != nil {
		return map[string]string{}, errors.Wrapf(err, "Image name: %s", in)
	}

	return Labels(sv), nil
}

// GetArtifact implements sous.Registry.GetArtifact.
func (nc *NameCache) GetArtifact(sid sous.SourceID) (*sous.BuildArtifact, error) {
	name, qls, err := nc.getImageName(sid)
	if err != nil {
		return nil, err
	}
	return NewBuildArtifact(name, qls), nil
}

func meansBodyUnchanged(err error) bool {
	_, ok := err.(NotModifiedErr)
	return ok || err == distribution.ErrManifestNotModified
}

// GetSourceID looks up the source ID for a given image name.
//  xxx consider un-exporting
func (nc *NameCache) GetSourceID(a *sous.BuildArtifact) (sous.SourceID, error) {
	in := a.Name
	var sid sous.SourceID

	nc.log("Getting source ID for", logging.ExtraDebug1Level, in)

	etag, repo, offset, version, _, err := nc.dbQueryOnName(in)
	if nif, ok := err.(NoSourceIDFound); ok {
		nc.log("Error: no source ID found", logging.ExtraDebug1Level, nif, a)
	} else if err != nil {
		nc.log("GetSourceID error", logging.WarningLevel, err)
		return sous.SourceID{}, err
	} else {
		nc.log("Found", logging.ExtraDebug1Level, repo, offset, version, etag)

		sid, err = sous.NewSourceID(repo, offset, version)
		if err != nil {
			return sid, err
		}

		dockerRef, err := reference.Parse(in)

		if r, isRef := dockerRef.(reference.Digested); err == nil && isRef {
			nc.log("Image name has digest: using knows source ID", logging.DebugLevel, r, sid)
			return sid, nil
		}
	}

	md, err := nc.RegistryClient.GetImageMetadata(in, etag)
	nc.log("md and err", logging.ExtraDebug1Level, md, err)
	if meansBodyUnchanged(err) {
		nc.log("Image Name and SourceID", logging.ExtraDebug1Level, in, sid)
		return sid, nil
	}
	if err != nil {
		nc.log("No docker image found: "+err.Error(), logging.ExtraDebug1Level, in, sid, err)
		return sid, err
	}

	newSID, err := SourceIDFromLabels(md.Labels)
	if err != nil {
		nc.log("SourceIDFromLabels failed: "+err.Error(), logging.ExtraDebug1Level, in, sid, err)
		return sid, err
	}

	qualities := qualitiesFromLabels(md.Labels)

	fullCanon := nc.DockerRegistryHost + "/" + md.CanonicalName
	mirrored := false
	if md.Registry != nc.DockerRegistryHost {
		mirrored = true
		_, err := nc.RegistryClient.GetImageMetadata(fullCanon, md.Etag)
		if err != nil && !meansBodyUnchanged(err) {
			fullCanon = md.Registry + "/" + md.CanonicalName
			nc.log("Docker image not found, leaving as", logging.DebugLevel, md.CanonicalName, nc.DockerRegistryHost, fullCanon)
		}
	}

	nc.log("Recording with etag as canonical for", logging.ExtraDebug1Level, fullCanon, md.Etag, newSID)
	err = nc.dbInsert(newSID, fullCanon, md.Etag, qualities)
	if err != nil {
		nc.log("Err recording", logging.DebugLevel, fullCanon, err)
		return sid, err
	}

	names := []string{}
	for _, n := range md.AllNames {
		names = append(names, nc.DockerRegistryHost+"/"+n)
	}
	err = nc.dbAddNames(nc.DockerRegistryHost+"/"+md.CanonicalName, names)
	nc.log("Recorded additional names", logging.ExtraDebug1Level, md.AllNames, fullCanon, nc.DockerRegistryHost, err)
	if err != nil && mirrored {
		err = nc.dbAddNames(md.Registry+"/"+md.CanonicalName, names)
		nc.log("Recorded mirrored names", logging.ExtraDebug1Level, md.AllNames, md.Registry+"/"+md.CanonicalName, md.Registry, err)
	}

	reportTableMetrics(nc.Log, nc.DB)
	nc.log("Images name (updated Source ID:)", logging.DebugLevel, in, newSID)
	return newSID, err
}

// GetImageName returns the docker image name for a given source ID
func (nc *NameCache) getImageName(sid sous.SourceID) (string, strpairs, error) {
	messages.ReportLogFieldsMessage("Getting image name for", logging.ExtraDebug1Level, nc.Log, sid)
	name, qualities, err := nc.getImageNameFromCache(sid)
	if err == nil {
		// We got it from the cache first time.
		reportCacheHit(nc.Log, sid, name)

		return name, qualities, nil
	}
	if _, ok := errors.Cause(err).(NoImageNameFound); !ok {
		// We got a probable database error, give up.
		reportCacheError(nc.Log, sid, err)
		return "", nil, errors.Wrapf(err, "getting name from cache of %s", nc.DockerRegistryHost)
	}
	reportCacheMiss(nc.Log, sid, name)
	// The error was a NoImageNameFound.
	if name, qualities, err = nc.getImageNameAfterHarvest(sid); err != nil {
		// Failed even after a harvest, give up.
		return "", nil, errors.Wrapf(err, "getting image from cache after harvest from %s", nc.DockerRegistryHost)
	}
	return name, qualities, nil
}

func (nc *NameCache) getImageNameFromCache(sid sous.SourceID) (string, strpairs, error) {
	cn, _, qls, err := nc.dbQueryOnSourceID(sid)
	return cn, qls, err
}

func (nc *NameCache) getImageNameAfterHarvest(sid sous.SourceID) (string, strpairs, error) {
	if err := nc.warmupSingle(sid); err == nil {
		return nc.getImageNameFromCache(sid)
	}
	err := nc.harvest(sid.Location)
	if err == nil {
		return nc.getImageNameFromCache(sid)
	}
	messages.ReportLogFieldsMessage("getImageName: harvest err", logging.WarningLevel, nc.Log, err)
	return "", nil, err
}

func qualitiesFromLabels(lm map[string]string) []sous.Quality {
	advs, ok := lm[`com.opentable.sous.advisories`]
	if !ok {
		return []sous.Quality{}
	}
	var qs []sous.Quality
	for _, adv := range strings.Split(advs, `,`) {
		qs = append(qs, sous.Quality{Name: adv, Kind: "advisory"})
	}
	return qs
}

// GetCanonicalName returns the canonical name for an image given any known name
func (nc *NameCache) GetCanonicalName(in string) (string, error) {
	_, _, _, _, cn, err := nc.dbQueryOnName(in)
	messages.ReportLogFieldsMessage("Canonicalizing - got", logging.DebugLevel, nc.Log, in, cn, err)
	return cn, err
}

// Insert puts a given SourceID/image name pair into the name cache
// used by Builder at the moment to register after a build
func (nc *NameCache) Insert(sid sous.SourceID, in, etag string, qs []sous.Quality) error {
	err := nc.dbInsert(sid, in, etag, qs)
	reportTableMetrics(nc.Log, nc.DB)
	return err
}

/*Harvesting source location*/
//{
//"message": "{\"Dir\":\"nested/there\",\"Repo\":\"https://github.com/opentable/wackadoo\"}"
//}
//Fields: Repo,Dir,SourceLocation
//Types: SourceLocation,string

func (nc *NameCache) harvest(sl sous.SourceLocation) error {
	messages.ReportLogFieldsMessage("Harvesting source location", logging.ExtraDebug1Level, nc.Log, sl)
	repos, err := nc.dbQueryOnSL(sl)
	if err != nil {
		messages.ReportLogFieldsMessage("Err looking up repos for location - proceeding with guessed repo", logging.WarningLevel, nc.Log, sl, err)
		repos = []string{}
	}
	guessed := fullRepoName(nc.DockerRegistryHost, sl, "")
	knowGuess := false

	messages.ReportLogFieldsMessage("Attempting to harvest repos", logging.ExtraDebug1Level, nc.Log, repos)
	for _, r := range repos {
		if r == guessed {
			knowGuess = true
		}
		err := nc.Warmup(r)
		if err != nil {
			return err
		}
	}
	if !knowGuess {
		err := nc.Warmup(guessed)
		if err != nil {
			return err
		}
	}
	return nil
}

// DBConfig is a database configuration for a NameCache.
type DBConfig struct {
	Driver, Connection string
}

func (nc *NameCache) dumpRows(io io.Writer, tx *sql.Tx, sql string) {
	fmt.Fprintln(io, sql)
	rows, err := tx.Query(sql)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	w := &tabwriter.Writer{}
	w.Init(io, 2, 4, 2, ' ', 0)
	heads, err := rows.Columns()
	if err != nil {
		panic(err)
	}

	fmt.Fprintln(w, strings.Join(heads, "\t"))

	vals := make([]interface{}, len(heads))
	for i := range vals {
		vals[i] = new(string)
	}

	for rows.Next() {
		rows.Scan(vals...)
		for i, v := range vals {
			if i != len(vals)-1 {
				fmt.Fprintf(w, "%s\t", *(v.(*string)))
			} else {
				fmt.Fprintf(w, "%s\n", *(v.(*string)))
			}
		}
	}
	w.Flush()
	fmt.Fprintln(io, "")
}

func (nc *NameCache) dumpTx(io io.Writer, tx *sql.Tx) {
	nc.dumpRows(io, tx, "select * from docker_repo_name")
	nc.dumpRows(io, tx, "select * from docker_search_location")
	nc.dumpRows(io, tx, "select * from repo_through_location")
	nc.dumpRows(io, tx, "select * from docker_search_metadata")
	nc.dumpRows(io, tx, "select * from docker_search_name")
	nc.dumpRows(io, tx, "select * from docker_image_qualities")
}

func (nc *NameCache) dump(io io.Writer) {
	ctx := context.TODO()
	tx, err := nc.DB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelRepeatableRead, ReadOnly: false})
	if err != nil {
		return
	}
	defer tx.Rollback() // we commit before returning...

	nc.dumpTx(io, tx)
}

type tableMetrics struct {
	DB *sql.DB
}

func (tm tableMetrics) rowCount(table string, sink logging.MetricsSink) {
	row := tm.DB.QueryRow("select count(1) from " + table)
	var n int64
	row.Scan(&n)
	sink.UpdateSample("dbrows."+table, n)
}

func (tm tableMetrics) MetricsTo(sink logging.MetricsSink) {
	sink.UpdateSample("dbconns", int64(tm.DB.Stats().OpenConnections))
	tm.rowCount("docker_repo_name", sink)
	tm.rowCount("docker_search_location", sink)
	tm.rowCount("docker_search_metadata", sink)
	tm.rowCount("docker_search_name", sink)
	tm.rowCount("docker_image_qualities", sink)
}

func reportTableMetrics(logger logging.LogSink, db *sql.DB) {
	msg := tableMetrics{
		DB: db,
	}
	logging.Deliver(logger, msg)
}

func versionString(v semv.Version) string {
	return v.Format(semv.Complete)
}

func (nc *NameCache) log(msg string, lvl logging.Level, data ...interface{}) {
	log(nc.Log, msg, lvl, data...)
}
