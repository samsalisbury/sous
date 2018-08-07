package actions

import (
	"fmt"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/restful"
)

// Jenkins is used to issue the command to make a new Deployment current for it's SourceID.
type Jenkins struct {
	HTTPClient           restful.HTTPClient
	TargetManifestID     sous.ManifestID
	LogSink              logging.LogSink
	User                 sous.User
	DefaultJenkinsConfig map[string]string
	*config.Config
}

// mergeDefaults will take the metadata map, and compare with defaults and merge the two
func (sj *Jenkins) mergeDefaults(metadata map[string]string) map[string]string {
	return make(map[string]string)
}

// place holder for generating map
func (sj *Jenkins) generateFileFromMap(jenkinsConfig map[string]string) error {
	return nil
}

// Do implements Action on Jenkins.
func (sj *Jenkins) Do() error {

	//Grab metadata from current manifest
	//Merge with Defaults
	//Write out Jenkins
	//Push back metadata

	//for now going to assume metadata for Jenkins file is CI-SF located, can change this in future
	currentConfigMap := make(map[string]string)
	mani := sous.Manifest{}
	_, err := sj.HTTPClient.Retrieve("/manifest", sj.TargetManifestID.QueryMap(), &mani, nil)

	// Eventually will make this configuration data in config.yaml
	clusterWithJenkinsConfig := "CI-SF"

	if len(clusterWithJenkinsConfig) < 1 {
		messages.ReportLogFieldsMessageToConsole("Please specify the JenkinsConfigCluster variable in sous config", logging.WarningLevel, sj.LogSink)
		return fmt.Errorf("no config cluster specified")
	}

	if err != nil || mani.Deployments["CI-SF"].Metadata == nil {
		messages.ReportLogFieldsMessageWithIDs("Couldn't determine metadata for CI-SF", logging.WarningLevel, sj.LogSink, err)
	} else {
		currentConfigMap = mani.Deployments["CI-SF"].Metadata
	}

	jenkinsConfig := sj.mergeDefaults(currentConfigMap)

	messages.ReportLogFieldsMessageWithIDs("Merged Config Data", logging.ExtraDebug1Level, sj.LogSink, jenkinsConfig)

	return sj.generateFileFromMap(jenkinsConfig)
}
