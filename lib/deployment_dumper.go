package sous

import (
	"fmt"
	"io"
	"text/tabwriter"
)

func DumpDeployments(io io.Writer, ds Deployments) {
	w := &tabwriter.Writer{}
	w.Init(io, 2, 4, 2, ' ', 0)
	fmt.Fprintln(w, TabbedDeploymentHeaders())

	for _, d := range ds.Snapshot() {
		fmt.Fprintln(w, d.Tabbed())
	}
	w.Flush()
}
