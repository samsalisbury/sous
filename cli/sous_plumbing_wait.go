package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path"
	"time"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/samsalisbury/coaxer"
)

// SousPlumbingWait is a command that waits for a certain condition to be met
// before returning.
//
// If the condition is met, it returns a zero exit code, otherwise if it times
// or another unexpected status is met, it returns a nonzero exit code.
type SousPlumbingWait struct {
	DeployFilterFlags config.DeployFilterFlags
	StatusPoller      *sous.StatusPoller
	Config            *config.Config
}

func init() { PlumbingSubcommands["status"] = &SousPlumbingWait{} }

// Help implements Command on SousPlumbingWait.
func (*SousPlumbingWait) Help() string {
	return `reports the status of a given deployment`
}

// AddFlags implements cmdr.AddFlags on SousPlumbingWait.
func (spw *SousPlumbingWait) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &spw.DeployFilterFlags, DeployFilterFlagsHelp)
}

// RegisterOn implements Registrant on SousPlumbingWait.
func (spw *SousPlumbingWait) RegisterOn(psy Addable) {
	psy.Add(&spw.DeployFilterFlags)
}

// Execute implements cmdr.Executor on SousPlumbingWait.
func (spw *SousPlumbingWait) Execute(args []string) cmdr.Result {

	timeout := 5 * time.Minute
	deployID, ok := spw.DeployFilterFlags.SpecificDeployID()
	if !ok {
		return cmdr.UsageErrorf("Please specify both -repo and -cluster flags.")
	}

	if spw.DeployFilterFlags.Tag == "" {
		return cmdr.UsageErrorf("Please specify -tag flag.")
	}

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
		err := spw.pollDeployState(timeout, deployID, spw.DeployFilterFlags.Tag)
		if err == nil {
			break
		}
		log.Printf("Waiting, not done because: %s", err)
	}

	return cmdr.Success()
}

func (spw *SousPlumbingWait) pollDeployState(timeout time.Duration, deployID sous.DeployID, version string) error {
	c := coaxer.NewCoaxer()

	result := c.Coax(context.TODO(), func() (interface{}, error) {
		return spw.fetchDeployState(deployID)
	}, "get deploy states")

	if err := result.Err(); err != nil {
		return result.Err()
	}

	ds, ok := result.Value().(sous.DeployState)
	if !ok {
		return fmt.Errorf("programmer error, got a %T, want a sous.DeployStates", result.Value())
	}

	if ds.Status != sous.DeployStatusSucceeded {
		return fmt.Errorf("deploy status: %s", ds.Status)
	}
	return nil
}

func (spw *SousPlumbingWait) fetchDeployState(deployID sous.DeployID) (*sous.DeployState, error) {
	u := path.Join(spw.Config.Server, "status")
	response, err := http.Get(u)
	defer func() {
		if response.Body != nil {
			if err := response.Body.Close(); err != nil {
				log.Println(err)
			}
		}
	}()
	if err != nil {
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
