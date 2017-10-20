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
