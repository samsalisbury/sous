package sqlgen

import (
	"fmt"
	"strings"
	"time"

	"github.com/opentable/sous/util/logging"
)

type (
	sqlMessage struct {
		logging.CallerInfo
		logging.MessageInterval
		mainTable string
		dir       direction
		sql       string
		rowcount  int
		err       error
		val       valueList
	}

	direction uint

	valueList []interface{}
)

const (
	read direction = iota
	write
)

func (vl valueList) EachField(fn logging.FieldReportFn) {
	str := ""
	for _, v := range vl {
		str = fmt.Sprintf("%s %q", str, v)
	}
	fn(logging.Arguments, str[1:])
}

func (dir direction) String() string {
	if dir == write {
		return "write"
	}
	return "read"
}
func newSQLMessage(started time.Time, mainTable string, dir direction, sql string, rowcount int, err error) *sqlMessage {
	return &sqlMessage{
		CallerInfo:      logging.GetCallerInfo(logging.NotHere()),
		MessageInterval: logging.NewInterval(started, time.Now()),
		mainTable:       mainTable,
		dir:             dir,
		sql:             sql,
		rowcount:        rowcount,
		err:             err,
	}
}

func reportSQLMessage(log logging.LogSink, started time.Time, mainTable string, dir direction, sql string, rowcount int, err error, vals ...interface{}) {
	msg := newSQLMessage(started, mainTable, dir, sql, rowcount, err)
	msg.ExcludeMe()
	if len(vals) > 0 {
		msg.val = valueList(vals)
	}
	logging.Deliver(log, msg)
}

// DefaultLevel implements LogMessage on sqlMessage
func (msg *sqlMessage) DefaultLevel() logging.Level {
	if msg.err == nil {
		return logging.InformationLevel
	}
	return logging.WarningLevel
}

// Message implements LogMessage on sqlMessage
func (msg *sqlMessage) Message() string {
	if msg.err == nil {
		return "SQL query"
	}
	return msg.err.Error()
}

// EachField implements LogMessage on sqlMessage
func (msg *sqlMessage) EachField(fn logging.FieldReportFn) {
	fn("@loglov3-otl", logging.SousSql)
	msg.CallerInfo.EachField(fn)
	msg.MessageInterval.EachField(fn)
	fn("sous-sql-query", msg.sql)
	fn("sous-sql-rows", msg.rowcount)
	if msg.err != nil {
		fn("sous-sql-errreturned", msg.err.Error())
	}
	if msg.val != nil {
		msg.val.EachField(fn)
	}
}

// MetricsTo implements MetricsMessage on sqlMessage
func (msg *sqlMessage) MetricsTo(sink logging.MetricsSink) {
	msg.MessageInterval.TimeMetric(strings.Join([]string{msg.mainTable, msg.dir.String(), "time"}, "."), sink)
	if msg.err != nil {
		sink.IncCounter(strings.Join([]string{msg.mainTable, msg.dir.String(), "errs"}, "."), 1)
		return
	}
	sink.UpdateSample(strings.Join([]string{msg.mainTable, msg.dir.String(), "rows"}, "."), int64(msg.rowcount))
	sink.IncCounter(strings.Join([]string{msg.mainTable, msg.dir.String(), "count"}, "."), int64(msg.rowcount))
}
