package cli

import (
	"flag"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
)

// SousNewDeploy has the same interface as SousDeploy, but uses the new
// PUT /single-deployment endpoint to begin the deployment, and polls by
// watching the returned rectification URL.
type SousNewDeploy struct {
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	StateReader       sous.StateReader
	TargetManifestID  graph.TargetManifestID
	dryrunOption      string
	waitStable        bool

	HTTPClient graph.HTTPClient
}

func init() { TopLevelCommands["newdeploy"] = &SousNewDeploy{} }

const sousNewDeployHelp = `deploys a new version into a particular cluster

usage: sous newdeploy -cluster <name> -tag <semver>

sous deploy will deploy the version tag for this application in the named
cluster.
`

// Help returns the help string for this command.
func (sd *SousNewDeploy) Help() string { return sousNewDeployHelp }

// AddFlags adds the flags for sous init.
func (sd *SousNewDeploy) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sd.DeployFilterFlags, DeployFilterFlagsHelp)

	fs.BoolVar(&sd.waitStable, "wait-stable", true,
		"wait for the deploy to complete before returning (otherwise, use --wait-stable=false)")
	fs.StringVar(&sd.dryrunOption, "dry-run", "none",
		"prevent rectify from actually changing things - "+
			"values are none,scheduler,registry,both")
}

// Execute creates the new deployment.
func (sd *SousNewDeploy) Execute(args []string) cmdr.Result {

	m := sous.Manifest{}
	q := sd.TargetManifestID.QueryMap()
	_, err := smg.HTTPClient.Retrieve("./manifest", q, &m, nil)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	messages.ReportLogFieldsMessage("SousNewDeploy.Execute Retrieved",
		logging.ExtraDebug1Level, smg.LogSink, m)

	cluster := sd.DeployFilterFlags.Cluster
	deploySpec, ok := m.Deployments[cluster]
	if !ok {
		return cmdr.UsageErrorf(
			"Manifest %q has no deployment for %q.\n"+
				"Tip: first add a deployment for this cluster with 'sous manifest get/set'",
			m.ID(), cluster)
	}
	m.Deployments = nil // We only want the "header" of the manifest.

	body := server.SingleDeploymentBody{
		ManifestHeader:  m,
		body.DeploySpec: deploySpec,
	}

	smg.HTTPClient.Update()

}
