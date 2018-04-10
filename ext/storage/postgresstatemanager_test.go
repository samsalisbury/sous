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
	// it's a SQL db driver. This is how you do that.
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

func SetupTest(t *testing.T) *PostgresStateManagerSuite {
	var err error

	t.Helper()

	suite := &PostgresStateManagerSuite{
		t:          t,
		Assertions: assert.New(t),
		require:    require.New(t),
	}

	db := setupDB(t)

	sink, ctrl := logging.NewLogSinkSpy()
	suite.manager = NewPostgresStateManager(db, sink)

	suite.logs = ctrl

	port := "6543"
	if np, set := os.LookupEnv("PGPORT"); set {
		port = np
	}
	connstr := fmt.Sprintf("dbname=sous_test host=localhost user=postgres port=%s sslmode=disable", port)
	if suite.db, err = sql.Open("postgres", connstr); err != nil {
		suite.FailNow("Error establishing test-assertion DB connection.", "Error: %v", err)
	}
	return suite
}

func TestPostgresStateManagerWriteState_success(t *testing.T) {
	suite := SetupTest(t)

	s := exampleState()

	err := suite.manager.WriteState(s, testUser)
	if !suite.NoError(err) {
		suite.logs.DumpLogs(t)
		t.FailNow()
	}
	suite.Equal(int64(4), suite.pluckSQL("select count(*) from deployments"))

	assert.Len(t, suite.logs.CallsTo("Fields"), 13)
	message := suite.logs.CallsTo("Fields")[0].PassedArgs().Get(0).([]logging.EachFielder)
	// XXX This message deserves its own test
	logging.AssertMessageFieldlist(t, message, append(
		append(logging.StandardVariableFields, logging.IntervalVariableFields...), "call-stack-function", "sous-sql-query", "sous-sql-rows"),
		map[string]interface{}{
			"@loglov3-otl": logging.SousSql,
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

func (suite *PostgresStateManagerSuite) pluckSQL(sql string) interface{} {
	var v interface{}

	suite.t.Helper()

	row := suite.db.QueryRow(sql)
	err := row.Scan(&v)
	suite.require.NoError(err)

	return v
}
