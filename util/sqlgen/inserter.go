package sqlgen

import (
	"context"
	"database/sql"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
)

// DoNothing is useful as the conflict argument of inserter.Exec
const DoNothing = `on conflict do nothing`

// Upsert is useful as the conflict argument of inserter.Exec
const Upsert = `on conflict {{.Candidates}} do update set {{.NonCandidates}} = {{.NSNonCandidates "excluded"}}`

type (
	// An Inserter performs inserts on a database.
	Inserter interface {
		Exec(table string, conflict string, fn func(FieldSet)) error
	}

	inserter struct {
		ctx context.Context
		log logging.LogSink
		tx  *sql.Tx
	}
)

// NewInserter creates an inserter struct, that helps build and execute INSERT queries.
func NewInserter(ctx context.Context, log logging.LogSink, tx *sql.Tx) Inserter {
	return inserter{ctx: ctx, log: log, tx: tx}
}

// Exec triggers an insertion.
func (ins inserter) Exec(table string, conflict string, fn func(FieldSet)) error {
	fields := NewFieldset()
	fn(fields)

	if !fields.Potent() {
		return nil
	}
	start := time.Now()

	sql := fields.InsertSQL(table, conflict)
	_, err := ins.tx.ExecContext(ins.ctx, sql, fields.InsertValues()...)
	reportSQLMessage(ins.log, start, table, write, sql, fields.RowCount(), err, fields.InsertValues()...)

	return errors.Wrapf(err, "Executing %q", sql)
}

// SingleRow is a shorthand to allow you to insert single rows easily.
func SingleRow(rf func(RowDef)) func(FieldSet) {
	return func(fs FieldSet) { fs.Row(rf) }
}
