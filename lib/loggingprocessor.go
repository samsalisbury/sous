package sous

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
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
	return msg.pair.Kind().String() + " deployment diff"
}

func (msg *deployableMessage) EachField(f logging.FieldReportFn) {
	marshal := func(thing string, data interface{}) string {
		b, err := json.Marshal(data)
		if err != nil {
			return fmt.Sprintf("error marshalling %s: %v", thing, err)
		}
		return string(b)
	}

	failureStatsAsStrings := func(stats []int) string {
		strs := []string{}
		for _, stat := range stats {
			strs = append(strs, strconv.Itoa(stat))
		}
		return strings.Join(strs, ",")
	}

	deployableFields := func(prefix string, d *Deployable) {
		f(prefix+"-status", d.Status.String())
		f(prefix+"-clustername", d.Deployment.ClusterName)
		f(prefix+"-repo", d.Deployment.SourceID.Location.Repo)
		f(prefix+"-offset", d.Deployment.SourceID.Location.Dir)
		f(prefix+"-tag", d.Deployment.SourceID.Version.String())
		f(prefix+"-flavor", d.Deployment.Flavor)
		f(prefix+"-owners", strings.Join(d.Deployment.Owners.Slice(), ","))
		f(prefix+"-kind", string(d.Deployment.Kind))
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
		f(prefix+"-checkready-failurestatuses", failureStatsAsStrings(d.DeployConfig.Startup.CheckReadyFailureStatuses))
		f(prefix+"-checkready-uritimeout", d.DeployConfig.Startup.CheckReadyURITimeout)
		f(prefix+"-checkready-interval", d.DeployConfig.Startup.CheckReadyInterval)
		f(prefix+"-checkready-retries", d.DeployConfig.Startup.CheckReadyRetries)

		f(prefix+"-artifact-name", d.BuildArtifact.Name)
		f(prefix+"-artifact-type", d.BuildArtifact.Type)
		f(prefix+"-artifact-qualities", d.BuildArtifact.Qualities.String())
	}

	f("@loglov3-otl", "sous-deployment-diff")
	msg.callerInfo.EachField(f)
	f("sous-deployment-id", msg.pair.ID().String())
	f("sous-manifest-id", msg.pair.ID().ManifestID.String())
	f("sous-diff-disposition", msg.pair.Kind().String())
	if msg.pair.Kind() == ModifiedKind {
		f("sous-deployment-diffs", msg.pair.Diffs.String())
	}

	if msg.pair.Prior != nil {
		deployableFields("sous-prior", msg.pair.Prior)
	}
	if msg.pair.Post != nil {
		deployableFields("sous-post", msg.pair.Post)
	}
}

func (log loggingProcessor) HandleResolution(rez *DiffResolution) {
	spew.Dump("loggingProcessor", rez)
	msg := &diffRezMessage{
		resolution: rez,
		callerInfo: logging.GetCallerInfo("loggingprocessor"),
	}
	logging.Deliver(msg, log.ls)
}

type diffRezMessage struct {
	resolution *DiffResolution
	callerInfo logging.CallerInfo
}

func (msg diffRezMessage) DefaultLevel() logging.Level {
	return logging.WarningLevel
}

func (msg diffRezMessage) Message() string {
	return string(msg.resolution.Desc)
}

func (msg diffRezMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", "sous-diff-resolution")
	msg.callerInfo.EachField(f)
	f("sous-deployment-id", msg.resolution.DeploymentID.String())
	f("sous-manifest-id", msg.resolution.ManifestID.String())
	f("sous-resolution-description", string(msg.resolution.Desc))
	marshallable := buildMarshableError(msg.resolution.Error.error)
	f("sous-resolution-errortype", marshallable.Type)
	f("sous-resolution-errormessage", marshallable.String)
}
