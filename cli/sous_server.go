package cli

import (
	"flag"
	"io/ioutil"
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
	Sous              *Sous
	DeployFilterFlags config.DeployFilterFlags
	Log               *sous.LogSet

	*config.Config
	*graph.SousGraph

	flags struct {
		dryrun,
		// laddr is the listen address in the form [host]:port
		laddr,
		// gdmRepo is a repository to clone into config.SourceLocation
		// in the case that config.SourceLocation is empty.
		gdmRepo string
	}
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
	fs.StringVar(&ss.flags.dryrun, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
	fs.StringVar(&ss.flags.laddr, `listen`, `:80`, "The address to listen on, like '127.0.0.1:https'")
	fs.StringVar(&ss.flags.gdmRepo, "gdm-repo", "", "Git repo containing the GDM (cloned into config.SourceLocation)")
}

// RegisterOn adds the DeploymentConfig to the psyringe to configure the
// labeller and registrar.
func (ss *SousServer) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunOption(ss.flags.dryrun))
	psy.Add(&ss.DeployFilterFlags)
}

// Execute is part of the cmdr.Command interface(s).
func (ss *SousServer) Execute(args []string) cmdr.Result {
	if err := ensureGDMExists(ss.flags.gdmRepo, ss.Config.StateLocation, ss.Log.Info.Printf); err != nil {
		return EnsureErrorResult(err)
	}
	ss.Log.Info.Println("Starting scheduled GDM resolution.")

	var arWrapper struct {
		*sous.AutoResolver
	}
	ss.SousGraph.MustInject(&arWrapper)
	arWrapper.AutoResolver.Kickoff()

	ss.Log.Info.Printf("Sous Server v%s running at %s for %s", ss.Sous.Version, ss.flags.laddr, ss.DeployFilterFlags.Cluster)

	return EnsureErrorResult(server.RunServer(ss.SousGraph, ss.flags.laddr)) //always non-nil
}

func ensureGDMExists(repo, localPath string, log func(string, ...interface{})) error {
	s, err := os.Stat(localPath)
	if err == nil && s.IsDir() {
		files, err := ioutil.ReadDir(localPath)
		if err != nil {
			return err
		}
		if len(files) != 0 {
			// The directory exists and is not empty, do nothing.
			if repo != "" {
				log("not pulling repo %q; directory already exist and is not empty: %q", repo, localPath)
			}
			return nil
		}
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
