package actions

import (
	"fmt"
	"testing"
	"time"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/samsalisbury/semv"
)

func TestUpdateBeginMessage(t *testing.T) {
	msg := newUpdateBeginMessage(
		2,
		sous.SourceID{
			Location: sous.SourceLocation{
				Repo: "github.com/opentable/example",
				Dir:  "first",
			},
			Version: semv.MustParse("1.2.7"),
		},
		sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/opentable/example",
					Dir:  "first",
				},
				Flavor: "vanilla",
			},
			Cluster: "test-example",
		},
		sous.User{
			Name:  "John Doe",
			Email: "jdoe@example.com",
		},
		time.Now(),
	)

	fixedFields := []string{
		"@timestamp",
		"call-stack-file",
		"call-stack-line-number",
		"call-stack-function",
		"thread-name",
		"started-at",
	}

	variableFields := map[string]interface{}{
		"@loglov3-otl": "sous-update-v1",
		"source-id":    "github.com/opentable/example,1.2.7,first",
		"deploy-id":    "test-example:github.com/opentable/example,first~vanilla",
		"user-email":   "jdoe@example.com",
		"try-number":   2,
	}

	logging.AssertMessageFields(t, msg, fixedFields, variableFields)
}

func TestUpdateSuccessMessage(t *testing.T) {
	msg := newUpdateSuccessMessage(
		2,
		sous.SourceID{
			Location: sous.SourceLocation{
				Repo: "github.com/opentable/example",
				Dir:  "first",
			},
			Version: semv.MustParse("1.2.7"),
		},
		sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/opentable/example",
					Dir:  "first",
				},
				Flavor: "vanilla",
			},
			Cluster: "test-example",
		},
		sous.User{
			Name:  "John Doe",
			Email: "jdoe@example.com",
		},
		time.Now(),
	)

	fixedFields := []string{
		"@timestamp",
		"call-stack-file",
		"call-stack-line-number",
		"call-stack-function",
		"thread-name",
		"started-at",
		"finished-at",
	}

	variableFields := map[string]interface{}{
		"@loglov3-otl": "sous-update-v1",
		"source-id":    "github.com/opentable/example,1.2.7,first",
		"deploy-id":    "test-example:github.com/opentable/example,first~vanilla",
		"user-email":   "jdoe@example.com",
		"try-number":   2,
	}

	logging.AssertMessageFields(t, msg, fixedFields, variableFields)
}

func TestUpdateErrorMessage(t *testing.T) {
	msg := newUpdateErrorMessage(
		2,
		sous.SourceID{
			Location: sous.SourceLocation{
				Repo: "github.com/opentable/example",
				Dir:  "first",
			},
			Version: semv.MustParse("1.2.7"),
		},
		sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/opentable/example",
					Dir:  "first",
				},
				Flavor: "vanilla",
			},
			Cluster: "test-example",
		},
		sous.User{
			Name:  "John Doe",
			Email: "jdoe@example.com",
		},
		time.Now(),
		fmt.Errorf("everything is on fire"),
	)

	fixedFields := []string{
		"@timestamp",
		"call-stack-file",
		"call-stack-line-number",
		"call-stack-function",
		"thread-name",
		"started-at",
	}

	variableFields := map[string]interface{}{
		"@loglov3-otl": "sous-update-v1",
		"source-id":    "github.com/opentable/example,1.2.7,first",
		"deploy-id":    "test-example:github.com/opentable/example,first~vanilla",
		"user-email":   "jdoe@example.com",
		"try-number":   2,
		"error":        "everything is on fire",
	}

	logging.AssertMessageFields(t, msg, fixedFields, variableFields)
}
