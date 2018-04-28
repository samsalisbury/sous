package sous

import "database/sql"

// A MaybeDatabase maybe has a DB in it.
type MaybeDatabase struct {
	Db  *sql.DB
	Err error
}
