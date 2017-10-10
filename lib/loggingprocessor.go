package sous

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opentable/sous/util/logging"
)

// Log adds a logging pipeline step onto a DeployableChans
func (d *DeployableChans) Log(ctx context.Context, ls logging.LogSink) *DeployableChans {
	proc := loggingProcessor{ls: ls}
	return d.Pipeline(ctx, proc)
}

type loggingProcessor struct {
	ls logging.LogSink
}

type deployableMessage struct {
	pair       *DeployablePair
	callerInfo logging.CallerInfo
}

func (log loggingProcessor) Start(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	log.doLog(dp)
	return dp, nil
}

func (log loggingProcessor) Stop(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	log.doLog(dp)
	return dp, nil
}

func (log loggingProcessor) Stable(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	log.doLog(dp)
	return dp, nil
}

func (log loggingProcessor) Update(dp *DeployablePair) (*DeployablePair, *DiffResolution) {
	log.doLog(dp)
	return dp, nil
}

func (log loggingProcessor) doLog(dp *DeployablePair) {
	msg := &deployableMessage{
		pair:       dp,
		callerInfo: logging.GetCallerInfo("loggingprocessor"),
	}

	logging.Deliver(msg, log.ls)
}

func (msg *deployableMessage) DefaultLevel() logging.Level {
	if msg.pair.Post == nil {
		return logging.WarningLevel
	}

	if msg.pair.Prior == nil {
		return logging.InformationLevel
	}

	if len(msg.pair.Diffs) == 0 {
		return logging.DebugLevel
	}

	return logging.InformationLevel
}

func (msg *deployableMessage) Message() string {
	if msg.pair.Prior == nil {
		return fmt.Sprintf("New deployment: %q", msg.pair.ID())
	}

	if msg.pair.Post == nil {
		return fmt.Sprintf("Deleted deployment: %q", msg.pair.ID())
	}

	if len(msg.pair.Diffs) == 0 {
		return fmt.Sprintf("Unchanged deployment: %q", msg.pair.ID())
	}

	return fmt.Sprintf("Modified deployment: %q (% #v)", msg.pair.ID(), msg.pair.Diffs)
}

func (msg *deployableMessage) EachField(f logging.FieldReportFn) {
	marshal := func(thing string, data interface{}) string {
		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Sprintf("error marshalling %s: %v", thing, err)
		}
		return string(b)
	}
	deployableFields := func(prefix string, d *Deployable) {
		f(prefix+"-status", d.Status.String())
		f(prefix+"-clustername", d.Deployment.ClusterName)
		f(prefix+"-repo", d.Deployment.SourceID.Location.Repo)
		f(prefix+"-offset", d.Deployment.SourceID.Location.Dir)
		f(prefix+"-offset", d.Deployment.SourceID.Version.String())
		f(prefix+"-flavor", d.Deployment.Flavor)
		f(prefix+"-owners", strings.Join(d.Deployment.Owners.Slice(), ","))
		f(prefix+"-kind", d.Deployment.Kind)
		f(prefix+"-resources", marshal("resources", d.DeployConfig.Resources))
		f(prefix+"-metadata", marshal("metadata", d.DeployConfig.Metadata))
		f(prefix+"-env", marshal("env", d.DeployConfig.Env))
		f(prefix+"-numinstances", d.DeployConfig.NumInstances)
		f(prefix+"-volumes", marshal("volumes", d.DeployConfig.Volumes))
		f(prefix+"-startup-skipcheck", d.DeployConfig.Startup.SkipCheck)
		f(prefix+"-startup-connectdelay", d.DeployConfig.Startup.ConnectDelay)
		f(prefix+"-startup-timeout", d.DeployConfig.Startup.Timeout)
		f(prefix+"-startup-connectinterval", d.DeployConfig.Startup.ConnectInterval)
		f(prefix+"-checkready-protocol", d.DeployConfig.Startup.CheckReadyProtocol)
		f(prefix+"-checkready-uripath", d.DeployConfig.Startup.CheckReadyURIPath)
		f(prefix+"-checkready-portindex", d.DeployConfig.Startup.CheckReadyPortIndex)
		f(prefix+"-checkready-failurestatuses", d.DeployConfig.Startup.CheckReadyFailureStatuses)
		f(prefix+"-checkready-uritimeout", d.DeployConfig.Startup.CheckReadyURITimeout)
		f(prefix+"-checkready-interval", d.DeployConfig.Startup.CheckReadyInterval)
		f(prefix+"-checkready-retries", d.DeployConfig.Startup.CheckReadyRetries)
	}

	msg.callerInfo.EachField(f)
	if msg.pair.Prior != nil {
		deployableFields("sous-prior", msg.pair.Prior)
	}

	if msg.pair.Post != nil {
		deployableFields("sous-post", msg.pair.Post)
	}
}
