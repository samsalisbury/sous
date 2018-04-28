package graph

import (
	sous "github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

func newMaybeDatabase(c LocalSousConfig) sous.MaybeDatabase {
	db, err := c.Database.DB()

	return sous.MaybeDatabase{Db: db, Err: errors.Wrapf(err, "%#v", c.Database)}
}
