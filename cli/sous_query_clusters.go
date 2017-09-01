package cli

import (
	"bytes"
	"flag"
	"fmt"
	"text/tabwriter"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

type (
	// SousQueryClusters is the description of the `sous query clusters` command.
	SousQueryClusters struct {
		graph.HTTPClient
		flags struct {
			includeURLs bool
		}
	}

	// copied from server - avoiding coupling to server implemention
	cluster struct {
		ClusterName string
		URL         string
	}

	serverListData struct {
		Servers []cluster
	}
)

func init() { QuerySubcommands["clusters"] = &SousQueryClusters{} }

const sousQueryClustersHelp = `The current set of available clusters for deployment.`

// Help prints the help
func (*SousQueryClusters) Help() string { return sousQueryClustersHelp }

// RegisterOn registers items on the DI graph
func (*SousQueryClusters) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&config.DeployFilterFlags{})
}

// AddFlags adds the flags for sous query clusters.
func (sqc *SousQueryClusters) AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&sqc.flags.includeURLs, "include-urls", false, "include the Sous URL for the cluster")
}

// Execute defines the behavior of `sous query gdm`
func (sqc *SousQueryClusters) Execute(args []string) cmdr.Result {
	clusters := &serverListData{}
	if _, err := sqc.Retrieve("./servers", nil, clusters, nil); err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	out := &bytes.Buffer{}
	w := &tabwriter.Writer{}
	w.Init(out, 2, 4, 2, ' ', 0)

	for _, s := range clusters.Servers {
		fmt.Fprintf(w, "%s", s.ClusterName)
		if sqc.flags.includeURLs {
			fmt.Fprintf(w, "\t%s", s.URL)
		}
		fmt.Fprintln(w, "")
	}
	w.Flush()

	return cmdr.SuccessData(out.Bytes())
}
