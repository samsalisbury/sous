package actions

import (
	"github.com/opentable/sous/cli"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
)

// Rectify processes a workstation rectify command. Mostly deprecated by sous server, but very useful when it is.
type Rectify struct {
	Resolver *sous.Resolver
	GDM      graph.CurrentGDM
	State    *sous.State
}

// Do implements Action on Rectify.
func (sr *Rectify) Do() error {
	if err := sr.Resolver.Begin(sr.GDM.Clone(), sr.State.Defs.Clusters).Wait(); err != nil {
		return err
	}

	return nil
}
