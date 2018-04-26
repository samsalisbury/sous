package sqlgen

import (
	"context"
	"database/sql"
	"time"

	"github.com/opentable/sous/util/logging"
)

// An Inserter performs inserts on a database.
type inserter struct {
	ctx context.Context
	log logging.LogSink
	tx  *sql.Tx
}

func NewInserter(ctx context.Context, log logging.LogSink, tx *sql.Tx) inserter {
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
	reportSQLMessage(ins.log, start, table, write, sql, fields.RowCount(), err)

	return err
}
