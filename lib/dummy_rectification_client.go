package sous

import (
	"log"
)

type (
	// DummyRectificationClient implements RectificationClient but doesn't act on the Mesos scheduler;
	// instead it collects the changes that would be performed and options
	DummyRectificationClient struct {
		logger   *log.Logger
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
func (drc *DummyRectificationClient) SetLogger(l *log.Logger) {
	l.Println("dummy begin")
	drc.logger = l
}

func (drc *DummyRectificationClient) log(v ...interface{}) {
	if drc.logger != nil {
		drc.logger.Print(v...)
	}
}

func (drc *DummyRectificationClient) logf(f string, v ...interface{}) {
	if drc.logger != nil {
		drc.logger.Printf(f, v...)
	}
}

// Deploy implements part of the RectificationClient interface
func (drc *DummyRectificationClient) Deploy(d Deployable, reqID string) error {
	drc.logf("Deploying instance %#v", d)
	drc.Deployed = append(drc.Deployed, d)
	return nil
}

// PostRequest (cluster, request id, instance count)
func (drc *DummyRectificationClient) PostRequest(d Deployable, id string) error {
	drc.logf("Creating application %#v", d, id)
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
