package sous

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/opentable/sous/util/logging"
)

type deployableMessage struct {
	pair       *DeployablePair
	callerInfo logging.CallerInfo
}

func (msg *deployableMessage) DefaultLevel() logging.Level {
	if msg.pair.Post == nil {
		return logging.WarningLevel
	}

	if msg.pair.Prior == nil {
		return logging.InformationLevel
	}

	if len(msg.pair.Diffs()) == 0 {
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

	deployConfigFields := func(prefix string, dc DeployConfig) {
		f(prefix+"-resources", marshal("resources", dc.Resources))
		f(prefix+"-metadata", marshal("metadata", dc.Metadata))
		f(prefix+"-env", marshal("env", dc.Env))
		f(prefix+"-numinstances", dc.NumInstances)
		f(prefix+"-volumes", marshal("volumes", dc.Volumes))
		f(prefix+"-startup-skipcheck", dc.Startup.SkipCheck)
		f(prefix+"-startup-connectdelay", dc.Startup.ConnectDelay)
		f(prefix+"-startup-timeout", dc.Startup.Timeout)
		f(prefix+"-startup-connectinterval", dc.Startup.ConnectInterval)
		f(prefix+"-checkready-protocol", dc.Startup.CheckReadyProtocol)
		f(prefix+"-checkready-uripath", dc.Startup.CheckReadyURIPath)
		f(prefix+"-checkready-portindex", dc.Startup.CheckReadyPortIndex)
		f(prefix+"-checkready-failurestatuses", failureStatsAsStrings(dc.Startup.CheckReadyFailureStatuses))
		f(prefix+"-checkready-uritimeout", dc.Startup.CheckReadyURITimeout)
		f(prefix+"-checkready-interval", dc.Startup.CheckReadyInterval)
		f(prefix+"-checkready-retries", dc.Startup.CheckReadyRetries)
	}

	buildArtifactFields := func(prefix string, ba *BuildArtifact) {
		if ba == nil {
			return
		}

		f(prefix+"-artifact-name", ba.Name)
		f(prefix+"-artifact-type", ba.Type)
		f(prefix+"-artifact-qualities", ba.Qualities.String())
	}

	deployableFields := func(prefix string, d *Deployable) {
		if d == nil {
			return
		}
		f(prefix+"-status", d.Status.String())
		f(prefix+"-clustername", d.Deployment.ClusterName)
		f(prefix+"-repo", d.Deployment.SourceID.Location.Repo)
		f(prefix+"-offset", d.Deployment.SourceID.Location.Dir)
		f(prefix+"-tag", d.Deployment.SourceID.Version.String())
		f(prefix+"-flavor", d.Deployment.Flavor)
		f(prefix+"-owners", strings.Join(d.Deployment.Owners.Slice(), ","))
		f(prefix+"-kind", string(d.Deployment.Kind))
		deployConfigFields(prefix, d.DeployConfig)
		buildArtifactFields(prefix, d.BuildArtifact)
	}

	f("@loglov3-otl", "sous-deployment-diff")
	msg.callerInfo.EachField(f)
	if msg.pair != nil {
		f("sous-deployment-id", msg.pair.ID().String())
		f("sous-manifest-id", msg.pair.ID().ManifestID.String())
		f("sous-diff-disposition", msg.pair.Kind().String())
		if msg.pair.Kind() == ModifiedKind {
			f("sous-deployment-diffs", msg.pair.Diffs().String())
		} else {
			f("sous-deployment-diffs", fmt.Sprintf("No detailed diff because pairwise diff kind is %q", msg.pair.Kind()))
		}

		if msg.pair.Prior != nil {
			deployableFields("sous-prior", msg.pair.Prior)
		}
		if msg.pair.Post != nil {
			deployableFields("sous-post", msg.pair.Post)
		}
	}
}
