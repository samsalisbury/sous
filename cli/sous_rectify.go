package cli

import (
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
)

// SousRectify is the injectable command object used for `sous rectify`
type SousRectify struct {
	Config       LocalSousConfig
	DockerClient LocalDockerClient
}

func init() { TopLevelCommands["rectify"] = &SousRectify{} }

const sousRectifyHelp = `
force Sous to make the deployment match the contents of a state directory

usage: sous rectify <dir>
`

// Help returns the help string
func (*SousRectify) Help() string { return sousRectifyHelp }

// Execute fulfills the cmdr.Executor interface
func (sr *SousRectify) Execute(args []string) cmdr.Result {
	if len(args) < 1 {
		return UsageErrorf("sous rectify requires a directory to load the intended deployment from")
	}
	dir := args[0]

	nc := sous.NewNameCache(sr.DockerClient, sr.Config.DatabaseDriver, sr.Config.DatabaseConnection)
	rc := sous.NewRectiAgent(nc)

	err := sous.ResolveFromDir(rc, dir)
	if err != nil {
		return EnsureErrorResult(err)
	}

	return Success()
}
