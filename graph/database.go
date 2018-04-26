package graph

import "database/sql"

type MaybeDatabase struct {
	Db  *sql.DB
	Err error
}

func newMaybeDatabase(c LocalSousConfig) MaybeDatabase {
	db, err := c.Database.DB()
	return MaybeDatabase{Db: db, Err: err}
}
