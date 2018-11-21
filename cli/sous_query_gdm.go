package cli

import (
	"flag"
	"fmt"

	"github.com/opentable/sous/cli/queries"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousQueryGDM is the description of the `sous query gdm` command
type SousQueryGDM struct {
	DeploymentQuery queries.Deployment
	flags           struct {
		filters string
		format  string
	}

	Out graph.OutWriter
	Err graph.ErrWriter
}

func init() { QuerySubcommands["gdm"] = &SousQueryGDM{} }

const sousQueryGDMHelp = `The intended state of deployment for every project and every cluster known to Sous.

The results of 'sous query gdm' and 'sous query ads' will not be identical if
a problem is preventing sous from modifying the current state of Singularity.
`

// Help prints the help
func (*SousQueryGDM) Help() string { return sousQueryGDMHelp }

// RegisterOn adds options set by flags to the injection graph.
func (*SousQueryGDM) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&config.DeployFilterFlags{})
}

// AddFlags adds the flags for 'sous query gdm'.
func (sb *SousQueryGDM) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&sb.flags.filters, "filters", "", "filter the output, space-separated list, e.g. 'hasimage=true zeroinstances=false hasowners=true'")
	fs.StringVar(&sb.flags.format, "format", "table", "output format, one of (table, json)")
}

func (sb *SousQueryGDM) dump(ds sous.Deployments) error {
	var err error
	switch sb.flags.format {
	default:
		err = cmdr.UsageErrorf("output format %q not valid, pick one of: table, json", sb.flags.format)
		fallthrough
	case "", "table":
		sous.DumpDeployments(sb.Out, ds)
	case "json":
		sous.JSONDeployments(sb.Out, ds)
	}
	return err
}

// Execute defines the behavior of `sous query gdm`.
func (sb *SousQueryGDM) Execute(args []string) cmdr.Result {
	af, err := sb.DeploymentQuery.ParseAttributeFilters(sb.flags.filters)
	if err != nil {
		return cmdr.UsageErrorf("parsing filters: %s", err)
	}

	filters := queries.DeploymentFilters{
		AttributeFilters: af,
	}

	result, err := sb.DeploymentQuery.Result(filters)
	if err != nil {
		return EnsureErrorResult(err)
	}

	fmt.Fprintf(sb.Err, "%d results\n", result.Deployments.Len())
	if err := sb.dump(result.Deployments); err != nil {
		return EnsureErrorResult(err)
	}
	return cmdr.Success()
}
