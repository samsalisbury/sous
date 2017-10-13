package actions

import (
	"github.com/opentable/sous/cli"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
)

type rectify struct {
	Resolver *sous.Resolver
	GDM      graph.CurrentGDM
	State    *sous.State
}

// GetRectify produces a rectify action.
func GetRectify(di injector, dryrun string, srcFlags cli.SourceFlags) Action {
	guardedAdd(di, "Dryrun", graph.DryrunOption(dryrun))
	guardedAdd(di, "SourceFlags", &srcFlags)

	r := &rectify{}
	di.Inject(r)
	return r
}

func (sr *rectify) Do() error {
	if err := sr.Resolver.Begin(sr.GDM.Clone(), sr.State.Defs.Clusters).Wait(); err != nil {
		return err
	}

	return nil
}
