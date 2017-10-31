package sous

import (
	"fmt"
	"testing"

	"github.com/opentable/sous/util/logging"
)

func TestPollerStartMessage(t *testing.T) {
	repo := "github.com/opentable/example"
	cluster := "test-cluster"

	poller := &StatusPoller{
		HTTPClient: nil,
		ResolveFilter: &ResolveFilter{
			Repo:    ResolveFieldMatcher{&repo},
			Cluster: ResolveFieldMatcher{&cluster},
		},
		User: User{Name: "Jane Doe", Email: "jdoe@example.com"},
		statePerCluster: map[string]*pollerState{
			"test-cluster": {
				LastResult: pollResult{
					url:       "sous.test-cluster.example.com",
					stat:      0,
					err:       nil,
					resolveID: "1234",
				},
				LastCycle: false,
			},
		},
		status: 0,
		/*
			logs:    nil,
			results: nil,
		*/
	}

	msg := newPollerStartMessage(poller)

	fixedFields := map[string]interface{}{
		"@loglov3-otl":    "sous-status-polling-v1",
		"user-name":       "Jane Doe",
		"user-email":      "jdoe@example.com",
		"filter-repo":     "github.com/opentable/example",
		"filter-cluster":  "test-cluster",
		"filter-tag":      "*",
		"filter-revision": "*",
		"filter-flavor":   "*",
		"filter-offset":   "*",
	}

	logging.AssertMessageFields(t, msg, logging.StandardVariableFields, fixedFields)
}
func TestPollerSubreportMessage(t *testing.T) {
	repo := "github.com/opentable/example"
	cluster := "test-cluster"

	poller := &StatusPoller{
		HTTPClient: nil,
		ResolveFilter: &ResolveFilter{
			Repo:    ResolveFieldMatcher{&repo},
			Cluster: ResolveFieldMatcher{&cluster},
		},
		User: User{Name: "Jane Doe", Email: "jdoe@example.com"},
		statePerCluster: map[string]*pollerState{
			"test-cluster": {
				LastResult: pollResult{
					url:       "sous.test-cluster.example.com",
					stat:      0,
					err:       nil,
					resolveID: "1234",
				},
				LastCycle: false,
			},
		},
		status: 0,
		/*
			logs:    nil,
			results: nil,
		*/
	}

	update := pollResult{
		url:       "sous.test-cluster.example.com",
		stat:      0,
		err:       nil,
		resolveID: "1234",
	}

	msg := newSubreportMessage(poller, update)

	fixedFields := map[string]interface{}{
		"@loglov3-otl":      "sous-polling-subresult-v1",
		"user-name":         "Jane Doe",
		"user-email":        "jdoe@example.com",
		"filter-repo":       "github.com/opentable/example",
		"filter-cluster":    "test-cluster",
		"filter-tag":        "*",
		"filter-revision":   "*",
		"filter-flavor":     "*",
		"filter-offset":     "*",
		"update-resolve-id": "1234",
		"update-url":        "sous.test-cluster.example.com",
		"update-status":     "ResolveNotPolled",
	}

	logging.AssertMessageFields(t, msg, logging.StandardVariableFields, fixedFields)
}

func TestPollerStatusMessage(t *testing.T) {
	repo := "github.com/opentable/example"
	cluster := "test-cluster"

	poller := &StatusPoller{
		HTTPClient: nil,
		ResolveFilter: &ResolveFilter{
			Repo:    ResolveFieldMatcher{&repo},
			Cluster: ResolveFieldMatcher{&cluster},
		},
		User: User{Name: "Jane Doe", Email: "jdoe@example.com"},
		statePerCluster: map[string]*pollerState{
			"test-cluster": {
				LastResult: pollResult{
					url:       "sous.test-cluster.example.com",
					stat:      0,
					err:       nil,
					resolveID: "1234",
				},
				LastCycle: false,
			},
		},
		status: 0,
	}

	msg := newPollerStatusMessage(poller, ResolveInProgress)

	fixedFields := map[string]interface{}{
		"@loglov3-otl":    "sous-status-polling-v1",
		"user-name":       "Jane Doe",
		"user-email":      "jdoe@example.com",
		"filter-repo":     "github.com/opentable/example",
		"filter-cluster":  "test-cluster",
		"filter-tag":      "*",
		"filter-revision": "*",
		"filter-flavor":   "*",
		"filter-offset":   "*",
		"deploy-status":   "ResolveNotPolled",
	}

	logging.AssertMessageFields(t, msg, logging.StandardVariableFields, fixedFields)
}

func TestPollerResolvedMessage(t *testing.T) {
	repo := "github.com/opentable/example"
	cluster := "test-cluster"

	poller := &StatusPoller{
		HTTPClient: nil,
		ResolveFilter: &ResolveFilter{
			Repo:    ResolveFieldMatcher{&repo},
			Cluster: ResolveFieldMatcher{&cluster},
		},
		User: User{Name: "Jane Doe", Email: "jdoe@example.com"},
		statePerCluster: map[string]*pollerState{
			"test-cluster": {
				LastResult: pollResult{
					url:       "sous.test-cluster.example.com",
					stat:      0,
					err:       nil,
					resolveID: "1234",
				},
				LastCycle: false,
			},
		},
		status: 0,
	}

	msg := newPollerResolvedMessage(poller, ResolveComplete, fmt.Errorf("not really an error just want some attention"))

	fixedFields := map[string]interface{}{
		"@loglov3-otl":    "sous-status-polling-v1",
		"user-name":       "Jane Doe",
		"user-email":      "jdoe@example.com",
		"filter-repo":     "github.com/opentable/example",
		"filter-cluster":  "test-cluster",
		"filter-tag":      "*",
		"filter-revision": "*",
		"filter-flavor":   "*",
		"filter-offset":   "*",
		"deploy-status":   "ResolveComplete",
	}

	logging.AssertMessageFields(t, msg, logging.StandardVariableFields, fixedFields)
}
