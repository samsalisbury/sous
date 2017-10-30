package sous

import (
	"fmt"
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestDeliversDiffMessages(t *testing.T) {
	spy, ctrl := logging.NewLogSinkSpy()
	proc := loggingProcessor{ls: spy}
	proc.doLog(&DeployablePair{})

	assert.Len(t, ctrl.CallsTo("LogMessage"), 1)
	// message for errors

	proc.HandleResolution(&DiffResolution{})
	assert.Len(t, ctrl.CallsTo("LogMessage"), 2)
}

func TestDiffMessages(t *testing.T) {
	msg := &deployableMessage{
		pair: &DeployablePair{
			Diffs: Differences{},
			Prior: deployableFixture(""),
			Post:  deployableFixture(""),
		},
		callerInfo: logging.GetCallerInfo(),
	}

	fixedFields := []string{
		"@timestamp",
		"call-stack-file",
		"call-stack-line-number",
		"call-stack-function",
		"thread-name",
	}

	fields := map[string]interface{}{
		"@loglov3-otl":          "sous-deployment-diff",
		"sous-deployment-id":    "test-cluster:github.com/opentable/example",
		"sous-diff-disposition": "same",
		"sous-manifest-id":      "github.com/opentable/example",

		"sous-prior-artifact-name":              "dockerhub.io/example:0.0.1",
		"sous-prior-artifact-qualities":         "",
		"sous-prior-artifact-type":              "docker",
		"sous-prior-checkready-failurestatuses": "",
		"sous-prior-checkready-interval":        0,
		"sous-prior-checkready-portindex":       0,
		"sous-prior-checkready-protocol":        "",
		"sous-prior-checkready-retries":         0,
		"sous-prior-checkready-uripath":         "",
		"sous-prior-checkready-uritimeout":      0,
		"sous-prior-clustername":                "test-cluster",
		"sous-prior-env":                        "{}",
		"sous-prior-flavor":                     "",
		"sous-prior-kind":                       "http-service",
		"sous-prior-metadata":                   "{}",
		"sous-prior-numinstances":               1,
		"sous-prior-offset":                     "",
		"sous-prior-owners":                     "",
		"sous-prior-repo":                       "github.com/opentable/example",
		"sous-prior-resources":                  "{\"cpus\":\"0.100\",\"memory\":\"356\",\"ports\":\"2\"}",
		"sous-prior-startup-connectdelay":       0,
		"sous-prior-startup-connectinterval":    0,
		"sous-prior-startup-skipcheck":          true,
		"sous-prior-startup-timeout":            0,
		"sous-prior-status":                     "DeployStatusActive",
		"sous-prior-tag":                        "0.0.1",
		"sous-prior-volumes":                    "[]",

		"sous-post-artifact-name":              "dockerhub.io/example:0.0.1",
		"sous-post-artifact-qualities":         "",
		"sous-post-artifact-type":              "docker",
		"sous-post-checkready-failurestatuses": "",
		"sous-post-checkready-interval":        0,
		"sous-post-checkready-portindex":       0,
		"sous-post-checkready-protocol":        "",
		"sous-post-checkready-retries":         0,
		"sous-post-checkready-uripath":         "",
		"sous-post-checkready-uritimeout":      0,
		"sous-post-clustername":                "test-cluster",
		"sous-post-env":                        "{}",
		"sous-post-flavor":                     "",
		"sous-post-kind":                       "http-service",
		"sous-post-metadata":                   "{}",
		"sous-post-numinstances":               1,
		"sous-post-offset":                     "",
		"sous-post-owners":                     "",
		"sous-post-repo":                       "github.com/opentable/example",
		"sous-post-resources":                  "{\"cpus\":\"0.100\",\"memory\":\"356\",\"ports\":\"2\"}",
		"sous-post-startup-connectdelay":       0,
		"sous-post-startup-connectinterval":    0,
		"sous-post-startup-skipcheck":          true,
		"sous-post-startup-timeout":            0,
		"sous-post-status":                     "DeployStatusActive",
		"sous-post-tag":                        "0.0.1",
		"sous-post-volumes":                    "[]",
	}

	msg.pair.name = msg.pair.Prior.ID()

	logging.AssertMessageFields(t, msg, fixedFields, fields)

	msg.pair.Post.Deployment.SourceID.Version = semv.MustParse("0.0.2")
	msg.pair.Diffs = Differences{"version not the same (this is the test message)"}

	fields["sous-deployment-diffs"] = "version not the same (this is the test message)"
	fields["sous-post-tag"] = "0.0.2"
	fields["sous-diff-disposition"] = "modified"

	logging.AssertMessageFields(t, msg, fixedFields, fields)
}

func TestDiffMessages_knownpanic(t *testing.T) {
	msg := &deployableMessage{
		pair: &DeployablePair{
			Diffs: Differences{},
			Prior: &Deployable{
				Status: 0,
				Deployment: &Deployment{
					DeployConfig: DeployConfig{
						Resources: map[string]string{
							"ports":  "3",
							"cpus":   "0.1",
							"memory": "1024",
						},
						Metadata: map[string]string{
							"": "",
						},
						Env: map[string]string{
							"OT_DISCO_INIT_URL:": "discovery-ci-uswest2.otenv.com",
						},
						NumInstances: 1,
						Volumes:      nil,
						Startup: Startup{
							SkipCheck:                 false,
							ConnectDelay:              10,
							Timeout:                   30,
							ConnectInterval:           1,
							CheckReadyProtocol:        "HTTP",
							CheckReadyURIPath:         "/health",
							CheckReadyPortIndex:       0,
							CheckReadyFailureStatuses: []int{500, 503},
							CheckReadyURITimeout:      5,
							CheckReadyInterval:        1,
							CheckReadyRetries:         120,
						},
					},
					ClusterName: "",
					Cluster:     nil,
					SourceID: SourceID{
						Location: SourceLocation{
							Repo: "github.com/opentable/consumer-service-xyz",
							Dir:  "",
						},
						Version: semv.MustParse("0.0.1"),
					},
					Flavor: "",
					Owners: map[string]struct{}{},
					Kind:   "",
				},
				BuildArtifact: nil,
			},
			Post: &Deployable{
				Status: 0,
				Deployment: &Deployment{
					DeployConfig: DeployConfig{
						Resources: map[string]string{
							"ports":  "3",
							"cpus":   "0.1",
							"memory": "1024",
						},
						Metadata: map[string]string{
							"": "",
						},
						Env: map[string]string{
							"OT_DISCO_INIT_URL:": "discovery-ci-uswest2.otenv.com",
						},
						NumInstances: 1,
						Volumes:      nil,
						Startup: Startup{
							SkipCheck:                 false,
							ConnectDelay:              10,
							Timeout:                   30,
							ConnectInterval:           1,
							CheckReadyProtocol:        "HTTP",
							CheckReadyURIPath:         "/health",
							CheckReadyPortIndex:       0,
							CheckReadyFailureStatuses: []int{500, 503},
							CheckReadyURITimeout:      5,
							CheckReadyInterval:        1,
							CheckReadyRetries:         120,
						},
					},
					ClusterName: "",
					Cluster:     nil,
					SourceID: SourceID{
						Location: SourceLocation{
							Repo: "github.com/opentable/consumer-service-xyz",
							Dir:  "",
						},
						Version: semv.MustParse("0.0.1"),
					},
					Flavor: "",
					Owners: map[string]struct{}{},
					Kind:   "",
				},
				BuildArtifact: nil,
			},
		},
	}

	fixedFields := []string{
		"@timestamp",
		"call-stack-file",
		"call-stack-line-number",
		"call-stack-function",
		"thread-name",
	}

	variableFields := map[string]interface{}{
		"@loglov3-otl":          "sous-deployment-diff",
		"sous-deployment-id":    ":",
		"sous-diff-disposition": "same",
		"sous-manifest-id":      "",

		"sous-post-checkready-failurestatuses": "500,503",
		"sous-post-checkready-interval":        1,
		"sous-post-checkready-portindex":       0,
		"sous-post-checkready-protocol":        "HTTP",
		"sous-post-checkready-retries":         120,
		"sous-post-checkready-uripath":         "/health",
		"sous-post-checkready-uritimeout":      5,
		"sous-post-clustername":                "",
		"sous-post-env":                        "{\"OT_DISCO_INIT_URL:\":\"discovery-ci-uswest2.otenv.com\"}",
		"sous-post-flavor":                     "",
		"sous-post-kind":                       "",
		"sous-post-metadata":                   "{\"\":\"\"}",
		"sous-post-numinstances":               1,
		"sous-post-offset":                     "",
		"sous-post-owners":                     "",
		"sous-post-repo":                       "github.com/opentable/consumer-service-xyz",
		"sous-post-resources":                  "{\"cpus\":\"0.1\",\"memory\":\"1024\",\"ports\":\"3\"}",
		"sous-post-startup-connectdelay":       10,
		"sous-post-startup-connectinterval":    1,
		"sous-post-startup-skipcheck":          false,
		"sous-post-startup-timeout":            30,
		"sous-post-status":                     "DeployStatusAny",
		"sous-post-tag":                        "0.0.1",
		"sous-post-volumes":                    "null",

		"sous-prior-checkready-failurestatuses": "500,503",
		"sous-prior-checkready-interval":        1,
		"sous-prior-checkready-portindex":       0,
		"sous-prior-checkready-protocol":        "HTTP",
		"sous-prior-checkready-retries":         120,
		"sous-prior-checkready-uripath":         "/health",
		"sous-prior-checkready-uritimeout":      5,
		"sous-prior-clustername":                "",
		"sous-prior-env":                        "{\"OT_DISCO_INIT_URL:\":\"discovery-ci-uswest2.otenv.com\"}",
		"sous-prior-flavor":                     "",
		"sous-prior-kind":                       "",
		"sous-prior-metadata":                   "{\"\":\"\"}",
		"sous-prior-numinstances":               1,
		"sous-prior-offset":                     "",
		"sous-prior-owners":                     "",
		"sous-prior-repo":                       "github.com/opentable/consumer-service-xyz",
		"sous-prior-resources":                  "{\"cpus\":\"0.1\",\"memory\":\"1024\",\"ports\":\"3\"}",
		"sous-prior-startup-connectdelay":       10,
		"sous-prior-startup-connectinterval":    1,
		"sous-prior-startup-skipcheck":          false,
		"sous-prior-startup-timeout":            30,
		"sous-prior-status":                     "DeployStatusAny",
		"sous-prior-tag":                        "0.0.1",
		"sous-prior-volumes":                    "null",
	}

	assert.NotPanics(t, func() {
		logging.AssertMessageFields(t, msg, fixedFields, variableFields)
	})
}

func TestDiffMessages_incomplete(t *testing.T) {
	msg := &deployableMessage{
		callerInfo: logging.GetCallerInfo(),
	}

	fixedFields := []string{
		"@timestamp",
		"call-stack-file",
		"call-stack-line-number",
		"call-stack-function",
		"thread-name",
	}

	variableFields := map[string]interface{}{
		"@loglov3-otl": "sous-deployment-diff",
	}

	logging.AssertMessageFields(t, msg, fixedFields, variableFields)

	msg.pair = &DeployablePair{}

	variableFields["sous-diff-disposition"] = "added"
	variableFields["sous-deployment-id"] = ":"
	variableFields["sous-manifest-id"] = ""

	logging.AssertMessageFields(t, msg, fixedFields, variableFields)
}

func TestDiffResolutionMessages(t *testing.T) {
	msg := &diffRezMessage{
		callerInfo: logging.GetCallerInfo(),
		resolution: &DiffResolution{
			DeploymentID: DeploymentID{
				ManifestID: ManifestID{
					Flavor: "", Source: SourceLocation{Repo: "github.com/opentable/example", Dir: ""},
				},
				Cluster: "test-cluster",
			},
			Desc:  ModifyDiff,
			Error: WrapResolveError(fmt.Errorf("dumb test error")),
		},
	}
	fixedFields := []string{
		"@timestamp",
		"call-stack-file",
		"call-stack-line-number",
		"call-stack-function",
		"thread-name",
	}

	logging.AssertMessageFields(t, msg, fixedFields, map[string]interface{}{
		"@loglov3-otl":                 "sous-diff-resolution",
		"sous-resolution-errortype":    "*errors.errorString",
		"sous-resolution-errormessage": "dumb test error",
		"sous-deployment-id":           "test-cluster:github.com/opentable/example",
		"sous-manifest-id":             "github.com/opentable/example",
		"sous-resolution-description":  "updated",
	})
}
