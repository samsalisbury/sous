package cli

import (
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousQueryArtifacts is the description of the `sous query gdm` command
type SousQueryArtifacts struct {
	*sous.RegistryDumper
	ErrWriter
}

func init() { QuerySubcommands["artifacts"] = &SousQueryArtifacts{} }

const sousQueryArtifactsHelp = `
Lists the images that Sous is currently aware of.

Note that Sous may discover more images after attempting a rectify

`

// Help prints the help
func (*SousQueryArtifacts) Help() string { return sousQueryArtifactsHelp }

// Execute defines the behavior of `sous query gdm`
func (sqa *SousQueryArtifacts) Execute(args []string) cmdr.Result {
	err := sqa.RegistryDumper.AsTable(sqa.ErrWriter)
	return ProduceResult(err)
}
