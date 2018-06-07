package storage

import (
	"database/sql"
	"fmt"
	"os"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	// It's a SQL db driver. This is how you do that.
	_ "github.com/lib/pq"
)

type PostgresStateManagerSuite struct {
	*assert.Assertions
	t       *testing.T
	require *require.Assertions
	manager *PostgresStateManager
	db      *sql.DB
	logs    logging.LogSinkController
}

func connstrForDBNamedf(nameFormat string, a ...interface{}) string {
	port := os.Getenv("PGPORT")
	if port == "" {
		port = "6543"
	}
	host := os.Getenv("PGHOST")
	if host == "" {
		host = "localhost"
	}
	name := fmt.Sprintf(nameFormat, a...)
	return fmt.Sprintf("host=%s port=%s dbname=%s user=postgres sslmode=disable", host, port, name)
}

func SetupTest(t *testing.T, name string) *PostgresStateManagerSuite {
	var err error

	t.Helper()

	suite := &PostgresStateManagerSuite{
		t:          t,
		Assertions: assert.New(t),
		require:    require.New(t),
	}

	db := sous.SetupDB(t)

	sink, ctrl := logging.NewLogSinkSpy()
	suite.manager = NewPostgresStateManager(db, sink)

	suite.logs = ctrl

	connstr := connstrForDBNamedf("test%s", name)
	if suite.db, err = sql.Open("postgres", connstr); err != nil {
		suite.FailNow("Error establishing test-assertion DB connection at %q.", "Error: %v", connstr, err)
	}
	if err := suite.db.Ping(); err != nil {
		suite.FailNow("Error checking test-assertion DB connection to %q.", "Error: %v", connstr, err)
	}
	return suite
}

func TestPostgresStateManagerWriteState_success(t *testing.T) {
	suite := SetupTest(t, "postgresstatemanagerwritestate_success") // because s/test//g
	defer sous.ReleaseDB(t)

	s := exampleState()

	err := suite.manager.WriteState(s, testUser)
	if !suite.NoError(err) {
		suite.logs.DumpLogs(t)
		t.FailNow()
	}
	suite.Equal(int64(4), suite.pluckSQL("select count(*) from deployments"))

	assert.Len(t, suite.logs.CallsTo("Fields"), 15)
	message := suite.logs.CallsTo("Fields")[0].PassedArgs().Get(0).([]logging.EachFielder)
	// XXX This message deserves its own test
	logging.AssertMessageFieldlist(t, message, append(
		append(logging.StandardVariableFields, logging.IntervalVariableFields...), "call-stack-function", "sous-sql-query", "sous-sql-rows"),
		map[string]interface{}{
			"@loglov3-otl":       logging.SousSql,
			"severity":           logging.InformationLevel,
			"call-stack-message": "SQL query",
		})

	suite.require.NoError(suite.manager.WriteState(s, testUser))
	// Want to be sure that the deployments history doesn't vacuously grow.
	if !suite.Equal(int64(4), suite.pluckSQL("select count(*) from deployments")) {
		rows, err := suite.db.Query("select * from deployments")
		suite.require.NoError(err)
		colNames, err := rows.Columns()
		suite.require.NoError(err)
		vals := make([]interface{}, len(colNames))
		valPtrs := make([]interface{}, len(colNames))
		for i := range vals {
			valPtrs[i] = &vals[i]
		}
		suite.t.Logf("%v", colNames)
		for rows.Next() {
			err := rows.Scan(valPtrs...)
			suite.require.NoError(err)
			suite.t.Logf("%v", vals)
		}
	}

	ns, err := suite.manager.ReadState()
	suite.require.NoError(err)

	assertSameClusters(t, s, ns)

	oldD, err := s.Deployments()
	suite.require.NoError(err)
	newD, err := ns.Deployments()
	suite.require.NoError(err)

	for diff := range oldD.Diff(newD).Pairs {
		switch diff.Kind() {
		default:
			suite.Fail("Difference detected between written and read states", "They are: %s %+#v", diff.Kind(), diff)
		case sous.ModifiedKind:
			suite.Fail("Difference detected between written and read states", "%+#v %+#v", diff, diff.Diffs())

		case sous.SameKind:
		}
	}
}

func assertSameClusters(t *testing.T, old *sous.State, new *sous.State) {
	ocs := old.Defs.Clusters
	ncs := new.Defs.Clusters

	onames := ocs.Names()
	nnames := ncs.Names()

	assert.ElementsMatch(t, onames, nnames)

	t.Logf("Cluster names: %q", onames)

	for _, n := range onames {
		oc, nc := ocs[n], ncs[n]

		assert.ElementsMatch(t, oc.AllowedAdvisories, nc.AllowedAdvisories)
		t.Logf("Cluster advisories: %q: %q", n, oc.AllowedAdvisories)
	}
}

func (suite *PostgresStateManagerSuite) pluckSQL(sql string) interface{} {
	var v interface{}

	suite.t.Helper()

	row := suite.db.QueryRow(sql)
	err := row.Scan(&v)
	suite.require.NoError(err)

	return v
}
