package actions

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/dto"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/server"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
	"github.com/pkg/errors"
	"github.com/vbauerster/mpb"
	"github.com/vbauerster/mpb/decor"
	"golang.org/x/crypto/ssh/terminal"
)

// Deploy is used to issue the command to make a new Deployment current for it's SourceID.
type Deploy struct {
	ResolveFilter      *sous.ResolveFilter
	StateReader        sous.StateReader
	HTTPClient         restful.HTTPClient
	TargetDeploymentID sous.DeploymentID
	LogSink            logging.LogSink
	User               sous.User
	Force, WaitStable  bool
	config.Config
}

// Do implements Action on Deploy.
func (sd *Deploy) Do() error {
	newVersion, err := sd.ResolveFilter.TagVersion()
	if err != nil {
		return err
	}

	d := server.SingleDeploymentBody{}
	q := sd.TargetDeploymentID.QueryMap()
	q["force"] = strconv.FormatBool(sd.Force)

	updater, err := sd.HTTPClient.Retrieve("./single-deployment", q, &d, nil)
	if err != nil {
		return errors.Errorf("Failed to retrieve current deployment: %s", err)
	}
	messages.ReportLogFieldsMessage("SousNewDeploy.Execute Retrieved Deployment",
		logging.ExtraDebug1Level, sd.LogSink, d)

	d.Deployment.Version = newVersion

	updateResponse, err := updater.Update(d, sd.User.HTTPHeaders())
	if err != nil {
		return cmdr.InternalErrorf("Failed to update deployment: %s", err)
	}

	if !sd.WaitStable {
		messages.ReportLogFieldsMessageToConsole(
			fmt.Sprintf("Deploy %q requested of server. Exiting optimistically.", sd.TargetDeploymentID),
			logging.DebugLevel,
			sd.LogSink,
		)
		return nil
	}

	if location := updateResponse.Location(); location != "" {
		fmt.Printf("Deployment queued: %s\n", location)
		pollTime := sd.Config.PollIntervalForClient

		logging.Deliver(sd.LogSink, logging.Console("\n"))

		var p *mpb.Progress
		var bar *mpb.Bar
		if terminal.IsTerminal(int(os.Stdin.Fd())) {
			p = mpb.New()
			// initialize bar with dynamic total and initial total guess = 80
			bar = p.AddBar(100,
				// indicate that total is dynamic
				mpb.BarDynamicTotal(),
				// trigger total auto increment by 1, when 18 % remains till bar completion
				mpb.BarAutoIncrTotal(18, 1),
				mpb.PrependDecorators(
					decor.CountersNoUnit("%d / %d", 12, 0),
				),
				mpb.AppendDecorators(
					decor.Percentage(5, 0),
				),
			)
		}

		result := sd.pollDeployQueue(location, pollTime, bar)

		if terminal.IsTerminal(int(os.Stdin.Fd())) && bar != nil && p != nil {
			bar.SetTotal(100, true)
			bar.Incr(100)
			bar.Complete()
			p.Wait()
			p.RemoveBar(bar)
		}
		return result
	}
	messages.ReportLogFieldsMessageToConsole(
		fmt.Sprintf("Desired version for %q already %q", sd.TargetDeploymentID, newVersion),
		logging.DebugLevel,
		sd.LogSink,
	)
	return nil
}

func timeTrack(start time.Time) string {
	elapsed := time.Since(start)
	return elapsed.String()
}

func (sd *Deploy) pollDeployQueue(location string, pollAtempts int, bar *mpb.Bar) error {
	start := time.Now()
	response := dto.R11nResponse{}
	location = "http://" + location

	for i := 0; i < pollAtempts; i++ {
		if bar != nil {
			bar.IncrBy(5)
		}
		if _, err := sd.HTTPClient.Retrieve(location, nil, &response, nil); err != nil {
			return cmdr.InternalErrorf("\n\tFailed to deploy: %s duration: %s\n", err, timeTrack(start))
		}

		queuePosition := response.QueuePosition

		if response.Resolution != nil && response.Resolution.Error != nil {
			return cmdr.InternalErrorf("\n\tFailed to deploy: %s duration: %s\n", response.Resolution.Error, timeTrack(start))
		}

		if queuePosition < 0 && response.Resolution != nil &&
			response.Resolution.DeployState != nil {

			if checkFinished(*response.Resolution) {
				if checkResolutionSuccess(*response.Resolution) {
					messages.ReportLogFieldsMessageToConsole(
						fmt.Sprintf("\n\tDeployment Complete %s, %s, duration: %s\n",
							response.Resolution.DeploymentID.String(), response.Resolution.DeployState.SourceID.Version, timeTrack(start)),
						logging.InformationLevel,
						sd.LogSink,
						logging.NewInterval(start, time.Now()),
					)
					return nil
				}
				//exit out to error handler
				return errors.Errorf("Failed to deploy %s: %s", location, response.Resolution.Error)
			}

		}
		time.Sleep(1 * time.Second)
	}

	responseJSON := ""
	if b, err := json.Marshal(response); err == nil {
		responseJSON = string(b)
	}

	return errors.Errorf("Failed to deploy %s after %d attempts for duration: %s\n Response: %s\n", location, pollAtempts, timeTrack(start), responseJSON)
}

func checkFinished(resolution sous.DiffResolution) bool {
	switch resolution.Desc {
	default:
		return false
	case sous.CreateDiff, sous.ModifyDiff:
		return true
	}
}

/*
const (
	// DeployStatusAny represents any deployment status.
0	DeployStatusAny DeployStatus = iota
	// DeployStatusPending means the deployment has been requested in the
	// cluster, but is not yet running.
1	DeployStatusPending
	// DeployStatusActive means the deployment is up and running.
2	DeployStatusActive
	// DeployStatusFailed means the deployment has failed.
3	DeployStatusFailed
)
For now treating everything but Active as return failed, could look to changin in future
*/
func checkResolutionSuccess(resolution sous.DiffResolution) bool {
	//We know 3 is a failure and 2 is a success so far
	switch resolution.DeployState.Status {
	default:
		return false
	case sous.DeployStatusActive:
		return true
	}
}
