package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/dto"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
	"golang.org/x/crypto/ssh/terminal"
)

// SousNewDeploy has the same interface as SousDeploy, but uses the new
// PUT /single-deployment endpoint to begin the deployment, and polls by
// watching the returned rectification URL.
type SousNewDeploy struct {
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	waitStable        bool
	force             bool
}

func init() { TopLevelCommands["newdeploy"] = &SousNewDeploy{} }

const sousNewDeployHelp = `deploys a new version into a particular cluster

usage: sous newdeploy [(options)]

sous newdeploy will deploy the version tag for this application in the named
cluster.

DEPRECATED: This now does the same thing as sous deploy, and this alias will be
removed in the future. Please update your scripts, documentation and habits
accordingly.`

// Help returns the help string for this command.
func (sd *SousNewDeploy) Help() string { return sousNewDeployHelp }

// AddFlags adds the flags for sous init.
func (sd *SousNewDeploy) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sd.DeployFilterFlags, NewDeployFilterFlagsHelp)

	fs.BoolVar(&sd.force, "force", false,
		"force deploy no matter if GDM already is at the correct version")
	fs.BoolVar(&sd.waitStable, "wait-stable", true,
		"wait for the deploy to complete before returning (otherwise, use --wait-stable=false)")
}

// Execute creates the new deployment.
func (sd *SousNewDeploy) Execute(args []string) cmdr.Result {
	deploy, err := ss.SousGraph.GetDeploy(sd.DeployFilterFlags, sd.force, sd.waitStable)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := deploy.Do(); err != nil {
		return EnsureErrorResult(err)
	}
	return cmdr.Success("Done.")
}
