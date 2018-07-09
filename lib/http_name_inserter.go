package sous

import (
	"net/url"
	"strings"
	"sync"

	multierror "github.com/hashicorp/go-multierror"
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
	serverList := ServerListData{}
	_, err := hni.client.Retrieve("./servers", nil, &serverList, nil)
	if err != nil {
		return err
	}

	bundle := map[string]restful.HTTPClient{}
	for _, s := range serverList.Servers {
		//messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Adding %s : %s", s.ClusterName, s.URL), logging.ExtraDebug1Level, hni.log, s)
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
				//TODO: Don't care about already existing (might be some other way to tell ./artifact that?)
				if !strings.Contains(err.Error(), "412 Precondition Failed") {
					errs <- err
				}
			}
		}(cl)
	}

	wg.Wait()

	var result *multierror.Error

	select {
	default:
	case err := <-errs:
		logging.ReportError(hni.log, err)
		result = multierror.Append(result, err)
	}

	return result.ErrorOrNil()
}

func simplifyQV(qvs url.Values) map[string]string {
	s := map[string]string{}
	for n, vs := range qvs {
		s[n] = vs[0]
	}
	return s
}
