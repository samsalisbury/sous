package sous

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/samsalisbury/semv"
)

func TestHTTPNameInserter(t *testing.T) {

	reqd := false

	srv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if meth := r.Method; strings.ToUpper(meth) != "PUT" {
			t.Errorf("Method should be PUT was: %s", meth)
		}
		if path := r.URL.Path; path != "/artifact" {
			t.Errorf("Path should be '/artifact' but was: %s", path)
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if len(body) <= 0 {
			t.Errorf("Empty body")
		}
		rw.WriteHeader(200)
		reqd = true
	}))

	startSrv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if meth := r.Method; strings.ToUpper(meth) != "GET" {
			t.Errorf("Method should be GET was %s", meth)
		}
		if path := r.URL.Path; path != "/servers" {
			t.Errorf("Path should be '/server' but was: %s", path)
		}
		rw.Header().Set("Content-Type", "application/json")
		sdata := serverListData{
			Servers: []server{{
				ClusterName: "test",
				URL:         srv.URL,
			}},
		}
		enc := json.NewEncoder(rw)
		enc.Encode(sdata)
	}))

	ls, _ := logging.NewLogSinkSpy()
	tid := TraceID("trace!")

	cl, err := restful.NewClient(startSrv.URL, ls, map[string]string{"OT-RequestId": string(tid)})
	if err != nil {
		t.Fatal(err)
	}

	hni := NewHTTPNameInserter(cl, tid, ls)
	if err != nil {
		t.Error(err)
	}
	err = hni.Insert(
		SourceID{Location: SourceLocation{Repo: "a-repo", Dir: "offset"}, Version: semv.MustParse("5.5.5")},
		BuildArtifact{
			DigestReference: "dockerthin.com/repo@sha256:012345678901234567890123456789AB012345678901234567890123456789AB",
			//Name:      "dockerthin.com/repo/latest",
			Type:      "docker",
			Qualities: []Quality{},
		},
	)
	if err != nil {
		t.Error(err)
	}
	if !reqd {
		t.Errorf("No request issued")
	}
}
