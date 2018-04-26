package graph

import "database/sql"

type maybeDatabase struct {
	db  *sql.DB
	err error
}

func newMaybeDatabase(c LocalSousConfig) maybeDatabase {
	db, err := c.Database.DB()
	return maybeDatabase{db: db, err: err}
}
