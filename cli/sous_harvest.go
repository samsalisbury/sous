package cli

import (
	"flag"
	"fmt"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousHarvest is the description of the `sous query gdm` command
type SousHarvest struct {
	State             *sous.State
	Registry          sous.Registry
	ErrWriter         graph.ErrWriter
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
}

func init() { TopLevelCommands["harvest"] = &SousHarvest{} }

const sousHarvestHelp = `Retrieve data from images in an artifact repostiory

usage: sous harvest -cluster <cluster> <repo>...
`

// Help prints the help
func (*SousHarvest) Help() string { return sousHarvestHelp }

// RegisterOn implements Registrar on SousHarvest
func (sh *SousHarvest) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&sh.DeployFilterFlags)
}

// AddFlags adds the -cluster flag.
func (sh *SousHarvest) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&sh.DeployFilterFlags.Cluster, "cluster", "", "cluster to harvest to")
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
