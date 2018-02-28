package sous

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/opentable/sous/util/logging"
)

type deploymentSubmessage struct {
	deployment *Deployment
	prefix     string
}

// NewDeploymentSubmessage returns an EachFielder that produces fields based on a Deployment
func NewDeploymentSubmessage(prefix string, dep *Deployment) logging.EachFielder {
	return &deploymentSubmessage{
		deployment: dep,
		prefix:     prefix,
	}
}

// EachField implements EachFielder on DeploymentSubmessage.
func (msg *deploymentSubmessage) EachField(f logging.FieldReportFn) {
	d := msg.deployment
	if d == nil {
		return
	}
	f(msg.prefix+"-clustername", d.ClusterName)
	f(msg.prefix+"-repo", d.SourceID.Location.Repo)
	f(msg.prefix+"-offset", d.SourceID.Location.Dir)
	f(msg.prefix+"-tag", d.SourceID.Version.String())
	f(msg.prefix+"-flavor", d.Flavor)
	f(msg.prefix+"-owners", strings.Join(d.Owners.Slice(), ","))
	f(msg.prefix+"-kind", string(d.Kind))

	msg.deployConfigFields(f)
}

func (msg *deploymentSubmessage) deployConfigFields(f logging.FieldReportFn) {
	dc := msg.deployment.DeployConfig

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
