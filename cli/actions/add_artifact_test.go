package actions

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/require"
)

func TestAddArtificact_Do(t *testing.T) {

	ls, _ := logging.NewLogSinkSpy()

	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "inspect")
	cctl.ResultSuccess("docker.otenv.com/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX", "")

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
	}))
	startSrv := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if meth := r.Method; strings.ToUpper(meth) != "GET" {
			t.Errorf("Method should be GET was %s", meth)
		}
		if path := r.URL.Path; path != "/servers" {
			t.Errorf("Path should be '/server' but was: %s", path)
		}
		rw.Header().Set("Content-Type", "application/json")
		sdata := sous.ServerListData{
			Servers: []sous.Server{{
				ClusterName: "test",
				URL:         srv.URL,
			}},
		}
		enc := json.NewEncoder(rw)
		enc.Encode(sdata)
	}))

	tid := sous.TraceID("trace!")

	cl, err := restful.NewClient(startSrv.URL, ls, map[string]string{"OT-RequestId": string(tid)})
	if err != nil {
		t.Fatal(err)
	}

	hni := sous.NewHTTPNameInserter(cl, tid, ls)
	if err != nil {
		t.Error(err)
	}
	a := &AddArtifact{
		LogSink:    ls,
		User:       sous.User{},
		Repo:       "github.com/testorg/repo",
		LocalShell: sh,
		Inserter:   sous.ClientInserter{Inserter: hni},
		Tag:        "0.0.3",
	}

	err = a.Do()

	require.NoError(t, err)
}
