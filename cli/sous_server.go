package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/git"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/shell"
)

// A SousServer represents the `sous server` command
type SousServer struct {
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

// Help is part of the cmdr.Command interface(s)
func (ss *SousServer) Help() string {
	return sousServerHelp
}

// AddFlags is part of the cmdr.Command interfaces(s)
func (ss *SousServer) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&ss.flags.laddr, `listen`, `:80`, "The address to listen on, like '127.0.0.1:https'")
	fs.StringVar(&ss.flags.gdmRepo, "gdm-repo", "", "Git repo containing the GDM (cloned into config.SourceLocation)")
}

// Execute is part of the cmdr.Command interface(s)
func (ss *SousServer) Execute(args []string) cmdr.Result {
	if err := ss.ensureGDMExists(ss.flags.gdmRepo, ss.Config.StateLocation); err != nil {
		return EnsureErrorResult(err)
	}
	ss.AutoResolver.Kickoff()
	err := server.RunServer(ss.Verbosity, ss.flags.laddr)
	return EnsureErrorResult(err) //always non-nil
}

func (ss *SousServer) ensureGDMExists(repo, localPath string) error {
	log := ss.Log.Info.Printf
	s, err := os.Stat(localPath)
	if err == nil {
		// The path exists, do nothing.
		if repo != "" {
			log("not pulling repo %q; state already exists at %q", repo, localPath)
		}
		return nil
	}
	log("got error %q", err)
	if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(localPath, 0777); err != nil {
		return err
	}
	s, err = os.Stat(localPath)
	if !s.IsDir() {
		return fmt.Errorf("%q exists and is not a directory", localPath)
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
