package storage

import (
	"strings"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

type sqlMessage struct {
	logging.CallerInfo
	logging.MessageInterval
	mainTable string
	dir       direction
	sql       string
	rowcount  int
	err       error
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

func reportSQLMessage(log logging.LogSink, started time.Time, mainTable string, dir direction, sql string, rowcount int, err error) {
	msg := newSQLMessage(started, mainTable, dir, sql, rowcount, err)
	msg.ExcludeMe()
	logging.Deliver(msg, log)
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

type (
	storeMessage struct {
		logging.CallerInfo
		logging.MessageInterval
		direction direction
		state     *sous.State
		err       error
	}

	direction uint
)

const (
	read direction = iota
	write
)

func reportReading(log logging.LogSink, started time.Time, state *sous.State, err error) {
	msg := newStoreMessage(started, read, state, err)
	msg.CallerInfo.ExcludeMe()
	logging.Deliver(msg, log)
}

func reportWriting(log logging.LogSink, started time.Time, state *sous.State, err error) {
	msg := newStoreMessage(started, write, state, err)
	msg.CallerInfo.ExcludeMe()
	logging.Deliver(msg, log)
}

func newStoreMessage(started time.Time, dir direction, state *sous.State, err error) *storeMessage {
	return &storeMessage{
		CallerInfo:      logging.GetCallerInfo(logging.NotHere()),
		MessageInterval: logging.NewInterval(started, time.Now()),
		state:           state,
		direction:       dir,
		err:             err,
	}
}

// DefaultLevel implements LogMessage on storeMessage.
func (msg *storeMessage) DefaultLevel() logging.Level {
	if msg.err == nil {
		return logging.DebugLevel
	}
	return logging.WarningLevel
}

// Message implements LogMessage on storeMessage.
func (msg *storeMessage) Message() string {
	return msg.direction.message()
}

func (dir direction) message() string {
	switch dir {
	default:
		return "Unknown state storage direction (shouldn't ever occur?)"
	case read:
		return "Reading state"
	case write:
		return "Writing state"
	}
}

func (dir direction) String() string {
	if dir == write {
		return "write"
	}
	return "read"
}

// EachField implements LogMessage on storeMessage.
func (msg *storeMessage) EachField(fn logging.FieldReportFn) {
	fn("@loglov3-otl", logging.SousGenericV1)
	msg.CallerInfo.EachField(fn)
	msg.MessageInterval.EachField(fn)
	if msg.err != nil {
		fn("sous-storage-error", msg.err.Error())
	}
	deps, err := msg.state.Deployments()
	if err == nil {
		fn("sous-storage-deployments", deps.Len())
	} else {
		fn("sous-storage-deployments", 0)
	}
}
