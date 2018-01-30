package sous

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/opentable/sous/util/logging"
)

// A DeployableSubmessage gathers the fields for logging for events
// which have a Deployable in their context.
type DeployableSubmessage struct {
	deployable *Deployable
	prefix     string
}

// NewDeployableSubmessage creates a new DeployableSubmessage.
func NewDeployableSubmessage(prefix string, dep *Deployable) *DeployableSubmessage {
	return &DeployableSubmessage{
		prefix:     prefix,
		deployable: dep,
	}
}

func (msg *DeployableSubmessage) deployConfigFields(f logging.FieldReportFn) {
	dc := msg.deployable.DeployConfig

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

	f(msg.prefix+"-resources", marshal("resources", dc.Resources))
	f(msg.prefix+"-metadata", marshal("metadata", dc.Metadata))
	f(msg.prefix+"-env", marshal("env", dc.Env))
	f(msg.prefix+"-numinstances", dc.NumInstances)
	f(msg.prefix+"-volumes", marshal("volumes", dc.Volumes))
	f(msg.prefix+"-startup-skipcheck", dc.Startup.SkipCheck)
	f(msg.prefix+"-startup-connectdelay", dc.Startup.ConnectDelay)
	f(msg.prefix+"-startup-timeout", dc.Startup.Timeout)
	f(msg.prefix+"-startup-connectinterval", dc.Startup.ConnectInterval)
	f(msg.prefix+"-checkready-protocol", dc.Startup.CheckReadyProtocol)
	f(msg.prefix+"-checkready-uripath", dc.Startup.CheckReadyURIPath)
	f(msg.prefix+"-checkready-portindex", dc.Startup.CheckReadyPortIndex)
	f(msg.prefix+"-checkready-failurestatuses", failureStatsAsStrings(dc.Startup.CheckReadyFailureStatuses))
	f(msg.prefix+"-checkready-uritimeout", dc.Startup.CheckReadyURITimeout)
	f(msg.prefix+"-checkready-interval", dc.Startup.CheckReadyInterval)
	f(msg.prefix+"-checkready-retries", dc.Startup.CheckReadyRetries)
}

func (msg *DeployableSubmessage) buildArtifactFields(f logging.FieldReportFn) {
	ba := msg.deployable.BuildArtifact

	if ba == nil {
		return
	}

	f(msg.prefix+"-artifact-name", ba.Name)
	f(msg.prefix+"-artifact-type", ba.Type)
	f(msg.prefix+"-artifact-qualities", ba.Qualities.String())
}

// EachField implements EachFielder on DeployableSubmessage.
func (msg *DeployableSubmessage) EachField(f logging.FieldReportFn) {
	d := msg.deployable
	if d == nil {
		return
	}
	f(msg.prefix+"-status", d.Status.String())
	f(msg.prefix+"-clustername", d.Deployment.ClusterName)
	f(msg.prefix+"-repo", d.Deployment.SourceID.Location.Repo)
	f(msg.prefix+"-offset", d.Deployment.SourceID.Location.Dir)
	f(msg.prefix+"-tag", d.Deployment.SourceID.Version.String())
	f(msg.prefix+"-flavor", d.Deployment.Flavor)
	f(msg.prefix+"-owners", strings.Join(d.Deployment.Owners.Slice(), ","))
	f(msg.prefix+"-kind", string(d.Deployment.Kind))

	msg.deployConfigFields(f)
	msg.buildArtifactFields(f)
}
