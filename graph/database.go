package graph

import (
	"database/sql"

	"github.com/pkg/errors"
)

type MaybeDatabase struct {
	Db  *sql.DB
	Err error
}

func newMaybeDatabase(c LocalSousConfig) MaybeDatabase {
	db, err := c.Database.DB()

	return MaybeDatabase{Db: db, Err: errors.Wrapf(err, "%#v", c.Database)}
}
