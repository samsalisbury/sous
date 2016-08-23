package cli

import (
	"fmt"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousHarvest is the description of the `sous query gdm` command
type SousHarvest struct {
	*sous.State
	sous.Registry
	ErrWriter
}

func init() { TopLevelCommands["harvest"] = &SousHarvest{} }

const sousHarvestHelp = `sous harvest <repo>...
Retrieve data from images in an artifact repostiory

`

// Help prints the help
func (*SousHarvest) Help() string { return sousHarvestHelp }

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
