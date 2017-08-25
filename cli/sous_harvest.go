package cli

import (
	"fmt"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousHarvest is the description of the `sous query gdm` command
type SousHarvest struct {
	*sous.State
	sous.Registry
	graph.ErrWriter
}

func init() { TopLevelCommands["harvest"] = &SousHarvest{} }

const sousHarvestHelp = `Retrieve data from images in an artifact repostiory

usage: sous harvest <repo>...
`

// Help prints the help
func (*SousHarvest) Help() string { return sousHarvestHelp }

func (*SousHarvest) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&config.DeployFilterFlags{})
}

// Execute defines the behavior of `sous query gdm`
func (sh *SousHarvest) Execute(args []string) cmdr.Result {
	if len(args) == 0 {
		return cmdr.EnsureErrorResult(fmt.Errorf("need an argument to harvest"))
	}

	var err error
	for _, repoName := range args {
		err = sh.Registry.Warmup(sh.State.Defs.DockerRepo + "/" + repoName)
		if err != nil {
			break
		}
	}
	return ProduceResult(err)
}
