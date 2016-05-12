package cli

import (
	"flag"

	"github.com/opentable/sous/util/cmdr"
)

type SousQueryAdc struct {
	Sous  *Sous
	flags struct {
		singularity string
		registry    string
	}
}

func init() { QuerySubcommands["adc"] = &SousQueryAdc{} }

const sousQueryAdcHelp = `
Queries the Singularity server and container registry to determine a synthetic global manifest.

This should resemble the manifest that was used to establish the current state of deployment.
`

func (*SousQueryAdc) Help() string { return sousBuildHelp }

func (sb *SousQueryAdc) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&sb.flags.singularity, "singularity", "",
		"the singularity server to query")
	fs.StringVar(&sb.flags.registry, "registry", "",
		"the container registry to query")
}

func (sb *SousQueryAdc) Execute(args []string) cmdr.Result {
	return InternalErrorf("not implemented")
}
