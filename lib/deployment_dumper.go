package sous

import (
	"fmt"
	"io"
	"text/tabwriter"
)

// DumpDeployments prints a bunch of Deployments to writer.
func DumpDeployments(writer io.Writer, ds Deployments) {
	w := &tabwriter.Writer{}
	w.Init(writer, 2, 4, 2, ' ', 0)

	fmt.Fprintln(w, TabbedDeploymentHeaders())

	for _, d := range ds.Snapshot() {
		fmt.Fprintln(w, d.Tabbed())
	}
	w.Flush()
}

// DumpDeployStatuses prints a bunch of DeployStates to writer.
func DumpDeployStatuses(writer io.Writer, ds DeployStates) {
	w := &tabwriter.Writer{}
	w.Init(writer, 2, 4, 2, ' ', 0)

	fmt.Fprintln(w, TabbedDeploymentHeaders())

	for _, d := range ds.Snapshot() {
		fmt.Fprintln(w, d.Tabbed())
	}
	w.Flush()
}
