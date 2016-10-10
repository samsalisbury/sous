package cli

import (
	"flag"
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/git"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/shell"
)

// A SousServer represents the `sous server` command.
type SousServer struct {
	Sous Sous
	*config.Verbosity
	*sous.AutoResolver
	Config graph.LocalSousConfig
	Log    *sous.LogSet
	flags  struct {
		// laddr is the listen address in the form [host]:port
		laddr,
		// gdmRepo is a repository to clone into config.SourceLocation
		// in the case that config.SourceLocation is empty.
		gdmRepo string
	}
}

func init() { TopLevelCommands["server"] = &SousServer{} }

const sousServerHelp = `
Runs the API server for Sous

usage: sous server
`

// Help is part of the cmdr.Command interface(s).
func (ss *SousServer) Help() string {
	return sousServerHelp
}

// AddFlags is part of the cmdr.Command interfaces(s).
func (ss *SousServer) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&ss.flags.laddr, `listen`, `:80`, "The address to listen on, like '127.0.0.1:https'")
	fs.StringVar(&ss.flags.gdmRepo, "gdm-repo", "", "Git repo containing the GDM (cloned into config.SourceLocation)")
}

// Execute is part of the cmdr.Command interface(s).
func (ss *SousServer) Execute(args []string) cmdr.Result {
	if err := ss.ensureGDMExists(ss.flags.gdmRepo, ss.Config.StateLocation); err != nil {
		return EnsureErrorResult(err)
	}
	ss.Log.Info.Println("Starting scheduled GDM resolution.")
	ss.AutoResolver.Kickoff()
	ss.Log.Info.Printf("Sous Server v%s running at %s", ss.Sous.Version, ss.flags.laddr)
	return EnsureErrorResult(server.RunServer(ss.Verbosity, ss.flags.laddr)) //always non-nil
}

func (ss *SousServer) ensureGDMExists(repo, localPath string) error {
	log := ss.Log.Info.Printf
	s, err := os.Stat(localPath)
	if err == nil && s.IsDir() {
		// The directory exists, do nothing.
		if repo != "" {
			log("not pulling repo %q; directory already exists: %q", repo, localPath)
		}
		return nil
	}
	if err := config.EnsureDirExists(localPath); err != nil {
		return EnsureErrorResult(err)
	}
	sh, err := shell.DefaultInDir(localPath)
	if err != nil {
		return EnsureErrorResult(err)
	}
	g, err := git.NewClient(sh)
	if err != nil {
		return EnsureErrorResult(err)
	}
	log("cloning %q into %q ...", repo, localPath)
	if err := g.CloneRepo(repo, localPath); err != nil {
		return EnsureErrorResult(err)
	}
	log("done")
	return nil
}
