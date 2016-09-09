package cli

import (
	"flag"
	"net"
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/cmdr"
)

// A SousServer represents the `sous server` command
type SousServer struct {
	*config.Verbosity
	flags struct {
		net, laddr string
	}
}

func init() { TopLevelCommands["server"] = &SousServer{} }

const sousServerHelp = `
Runs the API server for Sous

usage: sous server
`

// Help is part of the cmdr.Command interface(s)
func (ss *SousServer) Help() string {
	return sousServerHelp
}

// AddFlags is part of the cmdr.Command interfaces(s)
func (ss *SousServer) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&ss.flags.net, `net`, `tcp`, "Sets the kind of network: tcp,tcp4,tcp6,unix,unixpacket")
	fs.StringVar(&ss.flags.laddr, `listen`, `:80`, "The address to list on, like '127.0.0.1:https'")
}

// Execute is part of the cmdr.Command interface(s)
func (ss *SousServer) Execute(args []string) cmdr.Result {
	gf := func() *graph.SousGraph {
		g := graph.BuildGraph(os.Stdout, os.Stdout)
		g.Add(ss.Verbosity)
		return g
	}
	s := server.New(server.SousRouteMap, gf)
	l, err := net.Listen(ss.flags.net, ss.flags.laddr)
	if err != nil {
		return EnsureErrorResult(err)
	}
	err = s.Serve(l)
	return EnsureErrorResult(err) //always non-nil
}
