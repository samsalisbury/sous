package storage

import (
	"testing"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
)

func TestSQLMessageError(t *testing.T) {
	start := time.Now()
	spy, message := logging.AssertReport(t, func(log logging.LogSink) {
		reportSQLMessage(log, start, "test-table", write, "insert into test-table (x,y,z) = (1,2,3)", 1, errors.New("the database exploded"))
	})

	logging.AssertMessageFields(t, message, append(logging.StandardVariableFields, logging.IntervalVariableFields...), map[string]interface{}{
		"@loglov3-otl":         logging.SousSql,
		"call-stack-function":  "github.com/opentable/sous/ext/storage.TestSQLMessageError",
		"sous-sql-query":       "insert into test-table (x,y,z) = (1,2,3)",
		"sous-sql-rows":        1,
		"sous-sql-errreturned": "the database exploded",
	})

	assertMetricsCall(t, spy, "UpdateSample", "test-table.write.rows", 0)
	assertMetricsCall(t, spy, "IncCounter", "test-table.write.count", 0)
	assertMetricsCall(t, spy, "IncCounter", "test-table.write.errs", 1)
	assertMetricsCall(t, spy, "UpdateTimer", "test-table.write.time", 1)
}

func TestSQLMessageWrite(t *testing.T) {
	start := time.Now()
	spy, message := logging.AssertReport(t, func(log logging.LogSink) {
		reportSQLMessage(log, start, "test-table", write, "insert into test-table (x,y,z) = (1,2,3)", 1, nil)
	})

	logging.AssertMessageFields(t, message, append(logging.StandardVariableFields, logging.IntervalVariableFields...), map[string]interface{}{
		"@loglov3-otl":        logging.SousSql,
		"call-stack-function": "github.com/opentable/sous/ext/storage.TestSQLMessageWrite",
		"sous-sql-query":      "insert into test-table (x,y,z) = (1,2,3)",
		"sous-sql-rows":       1,
	})

	assertMetricsCall(t, spy, "UpdateSample", "test-table.write.rows", 1)
	assertMetricsCall(t, spy, "IncCounter", "test-table.write.count", 1)
	assertMetricsCall(t, spy, "IncCounter", "test-table.write.errs", 0)
	assertMetricsCall(t, spy, "UpdateTimer", "test-table.write.time", 1)
}

func TestSQLMessageRead(t *testing.T) {
	start := time.Now()
	spy, message := logging.AssertReport(t, func(log logging.LogSink) {
		reportSQLMessage(log, start, "test-table", read, "select * from test-table", 100, nil)
	})

	logging.AssertMessageFields(t, message, append(logging.StandardVariableFields, logging.IntervalVariableFields...), map[string]interface{}{
		"@loglov3-otl":        logging.SousSql,
		"call-stack-function": "github.com/opentable/sous/ext/storage.TestSQLMessageRead",
		"sous-sql-query":      "select * from test-table",
		"sous-sql-rows":       100,
	})

	assertMetricsCall(t, spy, "UpdateSample", "test-table.read.rows", 1)
	assertMetricsCall(t, spy, "IncCounter", "test-table.read.count", 1)
	assertMetricsCall(t, spy, "IncCounter", "test-table.read.errs", 0)
	assertMetricsCall(t, spy, "UpdateTimer", "test-table.read.time", 1)
}

func assertMetricsCall(t *testing.T, spy logging.LogSinkController, method, metric string, expectedCount int) {
	calls := spy.Metrics.CallsMatching(func(m string, args mock.Arguments) bool {
		if m != method {
			return false
		}

		if len(args) < 1 {
			return false
		}

		if args.String(0) != metric {
			return false
		}
		return true
	})
	if len(calls) != expectedCount {
		t.Errorf("Expected %d calls to %s, got %d", expectedCount, method, len(calls))
	}
}
