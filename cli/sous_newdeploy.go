package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/samsalisbury/semv"
)

// SousNewDeploy has the same interface as SousDeploy, but uses the new
// PUT /single-deployment endpoint to begin the deployment, and polls by
// watching the returned rectification URL.
type SousNewDeploy struct {
	DeployFilterFlags config.DeployFilterFlags `inject:"optional"`
	StateReader       graph.StateReader
	HTTPClient        *graph.ClusterSpecificHTTPClient
	TargetManifestID  graph.TargetManifestID
	LogSink           graph.LogSink
	dryrunOption      string
	waitStable        bool
	User              sous.User
}

func init() { TopLevelCommands["newdeploy"] = &SousNewDeploy{} }

const sousNewDeployHelp = `deploys a new version into a particular cluster

usage: sous newdeploy -cluster <name> -tag <semver>

EXPERIMENTAL COMMAND: This may or may not yet do what it says on the tin.
Feel free to try it out, but if it breaks, you get to keep both pieces.

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

// RegisterOn adds flag options to the graph.
func (sd *SousNewDeploy) RegisterOn(psy Addable) {
	psy.Add(graph.DryrunNeither)
	psy.Add(&sd.DeployFilterFlags)
}

type SingleDeployResponse struct {
	QueuePosition int
	Resolution    sous.DiffResolution
}

// Execute creates the new deployment.
func (sd *SousNewDeploy) Execute(args []string) cmdr.Result {

	cluster := sd.DeployFilterFlags.Cluster

	newVersion, err := semv.Parse(sd.DeployFilterFlags.Tag)
	if err != nil {
		return cmdr.UsageErrorf("not semver: -tag %s", sd.DeployFilterFlags.Tag)
	}

	d := server.SingleDeploymentBody{}
	q := sd.TargetManifestID.QueryMap()
	q["cluster"] = cluster
	updater, err := sd.HTTPClient.Retrieve("./single-deployment", q, &d, nil)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	messages.ReportLogFieldsMessage("SousNewDeploy.Execute Retrieved Deployment",
		logging.ExtraDebug1Level, sd.LogSink, d)

	d.Deployment.Version = newVersion

	updateResponse, err := updater.Update(d, sd.User.HTTPHeaders())
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	r := SingleDeployResponse{}
	if location := updateResponse.Location(); location != "" {
		//return cmdr.Successf("Deployment queued at: %s", location)
		fmt.Printf("deployment queued at : %s", location)
		//TODO LH at this point want to poll the url for the resolution
		resp, err := http.Get("http://" + location)
		if err != nil {
			// handle error
			fmt.Println("err : ", err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			fmt.Println("err : ", err)
			// fmt.Printf("body %s", body)
		} else {
			fmt.Printf("body : %s", body)
			//TODO LH this is not working the way I expect?
			json.Unmarshal(body, r)
			fmt.Printf("response : %s", r)
		}

	}
	return cmdr.Successf("Desired version for %q in cluster %q already %q",
		sd.TargetManifestID, cluster, sd.DeployFilterFlags.Tag)

}
