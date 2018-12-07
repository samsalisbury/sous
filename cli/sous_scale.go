package cli

import (
	"bytes"
	"flag"
	"io/ioutil"
	"strconv"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/restful"
	"github.com/pkg/errors"
)

// SousScale defines the `sous manifest edit` command.
type SousScale struct {
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	SousGraph         *graph.SousGraph
}

func init() { TopLevelCommands["scale"] = &SousScale{} }

const sousScaleHelp = "scale a sous deployment"

// Help implements Command on SousScale.
func (*SousScale) Help() string { return sousManifestEditHelp }

// AddFlags implements AddFlagger on SousScale.
func (ss *SousScale) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &ss.DeployFilterFlags, DeployFilterFlagsHelp)
}

// Execute implements Executor on SousScale.
func (ss *SousScale) Execute(args []string) cmdr.Result {

	if ss.DeployFilterFlags.Cluster == "" {
		return cmdr.UsageErrorf("-cluster flag required")
	}
	cluster := ss.DeployFilterFlags.Cluster

	if len(args) == 0 {
		return cmdr.UsageErrorf("argument required: <num-instances>")
	}
	if len(args) != 1 {
		return cmdr.UsageErrorf("exactly one argument required")
	}
	numStr := args[0]
	n, err := strconv.Atoi(numStr)
	if err != nil {
		return cmdr.UsageErrorf("argument %q not a decimal integer: %s", numStr, err)
	}
	if n < 0 {
		return cmdr.UsageErrorf("cannot scale to less than zero instances")
	}

	var up restful.Updater

	get, err := ss.SousGraph.GetManifestGet(ss.DeployFilterFlags, ioutil.Discard, &up)
	if err != nil {
		return EnsureErrorResult(err)
	}
	m, err := get.GetManifest()
	if err != nil {
		return EnsureErrorResult(errors.Wrapf(err, "getting manifest"))
	}

	d, ok := m.Deployments[cluster]
	if !ok {
		return cmdr.UsageErrorf("no deployment defined for cluster %q", cluster)
	}
	d.NumInstances = n
	m.Deployments[cluster] = d

	set, err := ss.SousGraph.GetManifestSet(ss.DeployFilterFlags, &up, &bytes.Buffer{})
	if err != nil {
		return EnsureErrorResult(err)
	}

	if err := set.SetManifest(m); err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success()
}
