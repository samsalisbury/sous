package actions

import (
	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

type PlumbNormalizeGDM struct {
	Log           logging.LogSink
	StateLocation string
	User          sous.User
}

func (p *PlumbNormalizeGDM) Do() error {

	dsm := storage.NewDiskStateManager(p.StateLocation, p.Log)

	state, err := dsm.ReadState()
	if err != nil {
		return err
	}
	if err := dsm.WriteState(state, p.User); err != nil {
		return err
	}

	return nil
}
