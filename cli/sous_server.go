package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// A SousServer represents the `sous server` command.
type SousServer struct {
	SousGraph         *graph.SousGraph
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	dryrun,
	laddr,
	gdmRepo string
	profiling bool
}

func init() { TopLevelCommands["server"] = &SousServer{} }

const sousServerHelp = `Runs the API server for Sous

usage: sous server
`

// Help is part of the cmdr.Command interface(s).
func (ss *SousServer) Help() string {
	return sousServerHelp
}

// AddFlags is part of the cmdr.Command interfaces(s).
func (ss *SousServer) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &ss.DeployFilterFlags, ClusterFilterFlagsHelp)
	fs.StringVar(&ss.dryrun, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
	fs.StringVar(&ss.laddr, `listen`, `:80`, "The address to listen on, like '127.0.0.1:https'")
	fs.StringVar(&ss.gdmRepo, "gdm-repo", "", "Git repo containing the GDM (cloned into config.SourceLocation)")
	fs.BoolVar(&ss.profiling, "profiling", false, "Enable profiling in the server.")
}

// Execute is part of the cmdr.Command interface(s).
func (ss *SousServer) Execute(args []string) cmdr.Result {
	server, err := ss.SousGraph.GetServer(ss.DeployFilterFlags, ss.dryrun, ss.laddr, ss.gdmRepo, ss.profiling)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := server.Do(); err != nil {
		return EnsureErrorResult(err)
	}
	return cmdr.Success("Done serving for Sous.")
}
