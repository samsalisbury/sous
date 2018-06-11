package sous

import (
	"net/url"
	"sync"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

type (
	// An HTTPNameInserter sends its inserts to the configured HTTP server
	HTTPNameInserter struct {
		tid     TraceID
		client  restful.HTTPClient
		clients map[string]restful.HTTPClient
		log     logging.LogSink
	}
)

// NewHTTPNameInserter creates a new HTTPNameInserter
func NewHTTPNameInserter(client restful.HTTPClient, tid TraceID, log logging.LogSink) *HTTPNameInserter {
	return &HTTPNameInserter{client: client, tid: tid, log: log}
}

func (hni *HTTPNameInserter) getClients() error {
	if hni.clients != nil {
		return nil
	}
	serverList := serverListData{}
	_, err := hni.client.Retrieve("./servers", nil, &serverList, nil)
	if err != nil {
		return err
	}

	bundle := map[string]restful.HTTPClient{}
	for _, s := range serverList.Servers {
		client, err := restful.NewClient(s.URL, hni.log.Child(s.ClusterName+".http-client"), map[string]string{"OT-RequestId": string(hni.tid)})
		if err != nil {
			return err
		}

		bundle[s.ClusterName] = client
	}
	hni.clients = bundle
	return nil
}

// Insert implements Inserter for HTTPNameInserter
func (hni *HTTPNameInserter) Insert(sid SourceID, ba BuildArtifact) error {
	if err := hni.getClients(); err != nil {
		return err
	}

	wg := sync.WaitGroup{}
	errs := make(chan error, len(hni.clients))
	for _, cl := range hni.clients {
		wg.Add(1)
		go func(client restful.HTTPClient) {
			defer wg.Done()
			if _, err := client.Create("./artifact", simplifyQV(sid.QueryValues()), ba, nil); err != nil {
				errs <- err
			}
		}(cl)
	}

	wg.Wait()
	select {
	default:
	case err := <-errs:
		logging.ReportError(hni.log, err)
	}
	return nil
}

func simplifyQV(qvs url.Values) map[string]string {
	s := map[string]string{}
	for n, vs := range qvs {
		s[n] = vs[0]
	}
	return s
}
