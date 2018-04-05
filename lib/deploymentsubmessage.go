package sous

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/constants"
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
		f(constants.FieldName("sous-unknown-type-type"), msg.prefix)
	case "sous-prior":
		f(constants.SousPriorClustername, d.ClusterName)
		f(constants.SousPriorRepo, d.SourceID.Location.Repo)
		f(constants.SousPriorOffset, d.SourceID.Location.Dir)
		f(constants.SousPriorTag, d.SourceID.Version.String())
		f(constants.SousPriorFlavor, d.Flavor)
		f(constants.SousPriorOwners, strings.Join(d.Owners.Slice(), ","))
		f(constants.SousPriorKind, string(d.Kind))
	case "sous-post":
		f(constants.SousPostClustername, d.ClusterName)
		f(constants.SousPostRepo, d.SourceID.Location.Repo)
		f(constants.SousPostOffset, d.SourceID.Location.Dir)
		f(constants.SousPostTag, d.SourceID.Version.String())
		f(constants.SousPostFlavor, d.Flavor)
		f(constants.SousPostOwners, strings.Join(d.Owners.Slice(), ","))
		f(constants.SousPostKind, string(d.Kind))
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
		f(constants.SousPriorResources, marshal("resources", dc.Resources))
		f(constants.SousPriorMetadata, marshal("metadata", dc.Metadata))
		f(constants.SousPriorEnv, marshal("env", dc.Env))
		f(constants.SousPriorNuminstances, dc.NumInstances)
		f(constants.SousPriorVolumes, marshal("volumes", dc.Volumes))
		f(constants.SousPriorStartupSkipcheck, dc.Startup.SkipCheck)
		f(constants.SousPriorStartupConnectdelay, dc.Startup.ConnectDelay)
		f(constants.SousPriorStartupTimeout, dc.Startup.Timeout)
		f(constants.SousPriorStartupConnectinterval, dc.Startup.ConnectInterval)
		f(constants.SousPriorCheckreadyProtocol, dc.Startup.CheckReadyProtocol)
		f(constants.SousPriorCheckreadyUripath, dc.Startup.CheckReadyURIPath)
		f(constants.SousPriorCheckreadyPortindex, dc.Startup.CheckReadyPortIndex)
		f(constants.SousPriorCheckreadyFailurestatuses, failureStatsAsStrings(dc.Startup.CheckReadyFailureStatuses))
		f(constants.SousPriorCheckreadyUritimeout, dc.Startup.CheckReadyURITimeout)
		f(constants.SousPriorCheckreadyInterval, dc.Startup.CheckReadyInterval)
		f(constants.SousPriorCheckreadyRetries, dc.Startup.CheckReadyRetries)
	case "sous-post":
		f(constants.SousPostResources, marshal("resources", dc.Resources))
		f(constants.SousPostMetadata, marshal("metadata", dc.Metadata))
		f(constants.SousPostEnv, marshal("env", dc.Env))
		f(constants.SousPostNuminstances, dc.NumInstances)
		f(constants.SousPostVolumes, marshal("volumes", dc.Volumes))
		f(constants.SousPostStartupSkipcheck, dc.Startup.SkipCheck)
		f(constants.SousPostStartupConnectdelay, dc.Startup.ConnectDelay)
		f(constants.SousPostStartupTimeout, dc.Startup.Timeout)
		f(constants.SousPostStartupConnectinterval, dc.Startup.ConnectInterval)
		f(constants.SousPostCheckreadyProtocol, dc.Startup.CheckReadyProtocol)
		f(constants.SousPostCheckreadyUripath, dc.Startup.CheckReadyURIPath)
		f(constants.SousPostCheckreadyPortindex, dc.Startup.CheckReadyPortIndex)
		f(constants.SousPostCheckreadyFailurestatuses, failureStatsAsStrings(dc.Startup.CheckReadyFailureStatuses))
		f(constants.SousPostCheckreadyUritimeout, dc.Startup.CheckReadyURITimeout)
		f(constants.SousPostCheckreadyInterval, dc.Startup.CheckReadyInterval)
		f(constants.SousPostCheckreadyRetries, dc.Startup.CheckReadyRetries)
	}
}
