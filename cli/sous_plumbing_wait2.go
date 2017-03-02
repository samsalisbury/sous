package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/temporary"
	"github.com/samsalisbury/coaxer"
	"github.com/samsalisbury/semv"
)

// SousPlumbingWait2 is the command description for `sous update`
type SousPlumbingWait2 struct {
	DeployFilterFlags config.DeployFilterFlags
	OTPLFlags         config.OTPLFlags
	Manifest          graph.TargetManifest
	GDM               graph.CurrentGDM
	State             *sous.State
	StateWriter       graph.StateWriter
	StateReader       graph.StateReader
	ResolveFilter     *graph.RefinedResolveFilter
	User              sous.User
	Config            *config.Config
}

func init() { PlumbingSubcommands["wait"] = &SousPlumbingWait2{} }

const sousPlumbingWaitHelp = `update the version to be deployed in a cluster

usage: sous update -cluster <name> [-tag <semver>] [-use-otpl-deploy|-ignore-otpl-deploy]

sous update will update the version tag for this application in the named
cluster. You can then use 'sous rectify' to have that version deployed.
`

// Help returns the help string for this command
func (spw *SousPlumbingWait2) Help() string { return sousPlumbingWaitHelp }

// AddFlags adds the flags for sous init.
func (spw *SousPlumbingWait2) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &spw.DeployFilterFlags, DeployFilterFlagsHelp)
}

// RegisterOn adds the DeploymentConfig to the psyringe to configure the
// labeller and registrar
func (spw *SousPlumbingWait2) RegisterOn(psy Addable) {
	psy.Add(&spw.DeployFilterFlags)
	psy.Add(&spw.OTPLFlags)
}

// Execute fulfills the cmdr.Executor interface.
func (spw *SousPlumbingWait2) Execute(args []string) cmdr.Result {
	sl := spw.Manifest.ID()
	sid, did, err := getIDs((*sous.ResolveFilter)(spw.ResolveFilter), sl)
	if err != nil {
		return EnsureErrorResult(err)
	}

	// Set 5m timeout for entire polling session.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	version := sid.Version

	server := spw.Config.Server
	if server == "" {
		return cmdr.UsageErrorf("Server required; use 'sous config server <url>' to set.")
	}

	for {
		// Use a promise to try to get current DeployStates from /status endpoint.
		// Then examine the DeployStates for the DeployID we are interested in.
		// If it is failed, return exit code 1.
		// It if it succeeded with the expected version, return exit code 0.
		// If it is succeeded with not the expected version, keep trying until
		// -timeout is reached.

		// Set 30s timeout per poll.
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		if err := spw.pollDeployState(ctx, did, version); err != nil {
			if !temporary.IsTemporary(err) {
				return cmdr.EnsureErrorResult(err)
			}
			log.Printf("Waiting, not done because: %s", err)
			time.Sleep(time.Second)
			continue
		}
		status := sous.InvalidDeployStatus
		sous.Log.Debug.Printf("Got deploy state %s for %q", status, did)
		return cmdr.Success()
	}
}

func (spw *SousPlumbingWait2) pollDeployState(ctx context.Context, deployID sous.DeployID, version semv.Version) error {
	c := coaxer.NewCoaxer()

	result := c.Coax(ctx, func() (interface{}, error) {
		return spw.fetchDeployState(deployID)
	}, "get deploy states")

	if err := result.Err(); err != nil {
		return result.Err()
	}

	ds, ok := result.Value().(*sous.DeployState)
	if !ok {
		return fmt.Errorf("programmer error, got a %T, want a *sous.DeployState", result.Value())
	}

	deployedVersion := ds.Deployment.SourceID.Version
	if !deployedVersion.Equals(version) {
		return temporary.Errorf("old version %q still deployed, awaiting %q", deployedVersion, version)
	}

	switch ds.Status {
	default:
		return fmt.Errorf("deploy reported an unknown status: %s", ds.Status)
	case sous.DeployStatusPending:
		return temporary.Errorf("deploy pending")
	case sous.DeployStatusFailed:
		return fmt.Errorf("deploy failed")
	case sous.DeployStatusSucceeded:
		return nil
	}
}

func (spw *SousPlumbingWait2) fetchDeployState(deployID sous.DeployID) (*sous.DeployState, error) {
	u := spw.Config.Server
	u = strings.TrimSuffix(u, "/")
	u = u + "/status"

	log.Printf("SPW: Getting: %s", u)
	response, err := http.Get(u)
	defer func() {
		if response != nil && response.Body != nil {
			if err := response.Body.Close(); err != nil {
				log.Println(err)
			}
		}
	}()
	if err != nil {
		log.Printf("SPW: HTTP Error getting %s: %s", u, err)
		return nil, err
	}
	if response.StatusCode != 200 {
		return nil, tempError{fmt.Errorf("got status code %d", response.StatusCode)}
	}
	if response.Body == nil {
		return nil, tempError{fmt.Errorf("got no body")}
	}

	responseBody := struct {
		DeployStates map[string]*sous.DeployState
	}{}
	ds := sous.NewDeployStates()
	if err := json.NewDecoder(response.Body).Decode(&responseBody); err != nil {
		return nil, err // decoding errors are not temporary
	}
	for _, d := range responseBody.DeployStates {
		ds.Add(d)
	}
	deployState, ok := ds.Get(deployID)
	if !ok {
		return nil, tempError{fmt.Errorf("no deployment with ID %q yet", deployID)}
	}

	return deployState, nil
}

type tempError struct {
	error
}

func (te *tempError) Temporary() bool {
	return true
}
