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

	switch msg.prefix {
	default:
		f(logging.FieldName("sous-unknown-type-type"), msg.prefix)
	case "sous-prior":
		f(logging.SousPriorClustername, d.ClusterName)
		f(logging.SousPriorRepo, d.SourceID.Location.Repo)
		f(logging.SousPriorOffset, d.SourceID.Location.Dir)
		f(logging.SousPriorTag, d.SourceID.Version.String())
		f(logging.SousPriorFlavor, d.Flavor)
		f(logging.SousPriorOwners, strings.Join(d.Owners.Slice(), ","))
		f(logging.SousPriorKind, string(d.Kind))
	case "sous-post":
		f(logging.SousPostClustername, d.ClusterName)
		f(logging.SousPostRepo, d.SourceID.Location.Repo)
		f(logging.SousPostOffset, d.SourceID.Location.Dir)
		f(logging.SousPostTag, d.SourceID.Version.String())
		f(logging.SousPostFlavor, d.Flavor)
		f(logging.SousPostOwners, strings.Join(d.Owners.Slice(), ","))
		f(logging.SousPostKind, string(d.Kind))
	}

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
	switch msg.prefix {
	case "sous-prior":
		f(logging.SousPriorResources, marshal("resources", dc.Resources))
		f(logging.SousPriorMetadata, marshal("metadata", dc.Metadata))
		f(logging.SousPriorEnv, marshal("env", dc.Env))
		f(logging.SousPriorNuminstances, dc.NumInstances)
		f(logging.SousPriorVolumes, marshal("volumes", dc.Volumes))
		f(logging.SousPriorStartupSkipcheck, dc.Startup.SkipCheck)
		f(logging.SousPriorStartupConnectdelay, dc.Startup.ConnectDelay)
		f(logging.SousPriorStartupTimeout, dc.Startup.Timeout)
		f(logging.SousPriorStartupConnectinterval, dc.Startup.ConnectInterval)
		f(logging.SousPriorCheckreadyProtocol, dc.Startup.CheckReadyProtocol)
		f(logging.SousPriorCheckreadyUripath, dc.Startup.CheckReadyURIPath)
		f(logging.SousPriorCheckreadyPortindex, dc.Startup.CheckReadyPortIndex)
		f(logging.SousPriorCheckreadyFailurestatuses, failureStatsAsStrings(dc.Startup.CheckReadyFailureStatuses))
		f(logging.SousPriorCheckreadyUritimeout, dc.Startup.CheckReadyURITimeout)
		f(logging.SousPriorCheckreadyInterval, dc.Startup.CheckReadyInterval)
		f(logging.SousPriorCheckreadyRetries, dc.Startup.CheckReadyRetries)
	case "sous-post":
		f(logging.SousPostResources, marshal("resources", dc.Resources))
		f(logging.SousPostMetadata, marshal("metadata", dc.Metadata))
		f(logging.SousPostEnv, marshal("env", dc.Env))
		f(logging.SousPostNuminstances, dc.NumInstances)
		f(logging.SousPostVolumes, marshal("volumes", dc.Volumes))
		f(logging.SousPostStartupSkipcheck, dc.Startup.SkipCheck)
		f(logging.SousPostStartupConnectdelay, dc.Startup.ConnectDelay)
		f(logging.SousPostStartupTimeout, dc.Startup.Timeout)
		f(logging.SousPostStartupConnectinterval, dc.Startup.ConnectInterval)
		f(logging.SousPostCheckreadyProtocol, dc.Startup.CheckReadyProtocol)
		f(logging.SousPostCheckreadyUripath, dc.Startup.CheckReadyURIPath)
		f(logging.SousPostCheckreadyPortindex, dc.Startup.CheckReadyPortIndex)
		f(logging.SousPostCheckreadyFailurestatuses, failureStatsAsStrings(dc.Startup.CheckReadyFailureStatuses))
		f(logging.SousPostCheckreadyUritimeout, dc.Startup.CheckReadyURITimeout)
		f(logging.SousPostCheckreadyInterval, dc.Startup.CheckReadyInterval)
		f(logging.SousPostCheckreadyRetries, dc.Startup.CheckReadyRetries)
	}
}
