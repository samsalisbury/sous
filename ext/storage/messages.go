package storage

import (
	"time"

	"github.com/opentable/sous/util/logging"
)

type sqlMessage struct {
	logging.CallerInfo
	logging.MessageInterval
	sql string
	err error
}

func newSQLMessage(started time.Time, sql string, err error) *sqlMessage {
	return &sqlMessage{
		CallerInfo:      logging.GetCallerInfo(logging.NotHere()),
		MessageInterval: logging.NewInterval(started, time.Now()),
		sql:             sql,
		err:             err,
	}
}

func reportSQLMessage(log logging.LogSink, started time.Time, sql string, err error) {
	msg := newSQLMessage(started, sql, err)
	msg.ExcludeMe()
	logging.Deliver(msg, log)
}

// DefaultLevel implements LogMessage on sqlMessage
func (msg *sqlMessage) DefaultLevel() logging.Level {
	return logging.InformationLevel
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
	msg.CallerInfo.EachField(fn)
	msg.MessageInterval.EachField(fn)
	fn("sous-sql-query", msg.sql)
	// sql.Rows are a cursor - they don't have a length...
	//fn("sous-sql-rowsreturned", msg.rows)
	if msg.err != nil {
		fn("sous-sql-errreturned", msg.err.Error())
	}
}
