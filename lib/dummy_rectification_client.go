package sous

import (
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
)

type (
	// DummyRectificationClient implements RectificationClient but doesn't act on the Mesos scheduler;
	// instead it collects the changes that would be performed and options
	DummyRectificationClient struct {
		logger   logging.LogSink
		Created  []Deployable
		Deployed []Deployable
		Deleted  []dummyDelete
	}

	dummyDelete struct {
		Cluster, Reqid, Message string
	}
)

// NewDummyRectificationClient builds a new DummyRectificationClient
func NewDummyRectificationClient() *DummyRectificationClient {
	return &DummyRectificationClient{}
}

// SetLogger sets the logger for the client
func (drc *DummyRectificationClient) SetLogger(l logging.LogSink) {
	drc.logger = l
	drc.logf("dummy begin")
}

func (drc *DummyRectificationClient) logf(f string, v ...interface{}) {
	if drc.logger != nil {
		messages.ReportLogFieldsMessage(f, logging.WarningLevel, drc.logger, v...)
	}
}

// Deploy implements part of the RectificationClient interface
func (drc *DummyRectificationClient) Deploy(d Deployable, reqID, depID string) error {
	drc.logf("Deploying instance %#v", d)
	drc.Deployed = append(drc.Deployed, d)
	return nil
}

// PostRequest (cluster, request id, instance count)
func (drc *DummyRectificationClient) PostRequest(d Deployable, id string) error {
	drc.logf("Creating application %#v %s", d, id)
	drc.Created = append(drc.Created, d)
	return nil
}

// DeleteRequest (cluster url, request id, instance count, message)
func (drc *DummyRectificationClient) DeleteRequest(
	cluster, reqid, message string) error {
	drc.logf("Deleting application %s %s %s", cluster, reqid, message)
	drc.Deleted = append(drc.Deleted, dummyDelete{cluster, reqid, message})
	return nil
}
