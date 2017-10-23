package actions

import (
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
)

// Rectify processes a workstation rectify command. Mostly deprecated by sous server, but very useful when it is.
type Rectify struct {
	Resolver *sous.Resolver
	State    *sous.State
	Log      logging.LogSink
}

// Do implements Action on Rectify.
func (sr *Rectify) Do() error {
	gdm, err := sr.State.Deployments()
	if err != nil {
		return err
	}

	if err := sr.Resolver.Begin(gdm, sr.State.Defs.Clusters).Wait(); err != nil {
		return err
	}

	return nil
}
