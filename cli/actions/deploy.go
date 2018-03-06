package actions

import (
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type Deploy struct {
	StateReader sous.StateReader
	Client      restful.HTTPClient
}

func (d *Deploy) Do() error {
	return nil
}
