package actions

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/require"
)

func TestAddArtifact_Do(t *testing.T) {

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
		Config:     &config.Config{},
	}

	err = a.Do()

	require.NoError(t, err)
}

func TestSelectDigest_ok(t *testing.T) {
	cases := []struct {
		configuredDockerReg string
		dockerInspectOutput string
		selectedRef         string
	}{
		{
			"docker.example.org",
			`
			['docker.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX']
			`,
			`docker.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX`,
		},
		{
			"docker.example.org",
			`
			=['docker.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX']
			`,
			`docker.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX`,
		},
		{
			"docker.example.org",
			`
			=['docker.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX']
			=['other.docker.reg/blahblagh:0.0.1@shaXYZ']
			 `,
			`docker.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX`,
		},
		{
			"docker.example.org",
			`
			=[docker.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX]
			=[other.docker.reg/blahblagh:0.0.1@shaXYZ]
			 `,
			`docker.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX`,
		},
		{
			"docker.example.org",
			`
			=[other.docker.reg/blahblagh:0.0.1@shaXYZ]
			=[docker.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX]
			`,
			`docker.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("", func(t *testing.T) {
			got, err := selectDigest(tc.dockerInspectOutput, tc.configuredDockerReg)
			if err != nil {
				t.Fatal(err)
			}
			want := tc.selectedRef
			if got != want {
				t.Errorf("got %q; want %q", got, want)
			}
		})
	}
}

func TestSelectDigest_err(t *testing.T) {
	cases := []struct {
		configuredDockerReg string
		dockerInspectOutput string
		wantErr             string
	}{
		{
			"docker.example.org",
			`
			['other.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX']
			`,
			`no digest for this image had registry "docker.example.org"`,
		},
		{
			"docker.example.org",
			`
			=['other.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX']
			`,
			`no digest for this image had registry "docker.example.org"`,
		},
		{
			"docker.example.org",
			`
			=['other.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX']
			=['other.docker.reg/blahblagh:0.0.1@shaXYZ']
			 `,
			`no digest for this image had registry "docker.example.org"`,
		},
		{
			"docker.example.org",
			`
			=[other.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX]
			=[other.docker.reg/blahblagh:0.0.1@shaXYZ]
			 `,
			`no digest for this image had registry "docker.example.org"`,
		},
		{
			"docker.example.org",
			`
			=[other.docker.reg/blahblagh:0.0.1@shaXYZ]
			=[other.example.org/hello:0.0.1@shaXXXXXXXXXXXXXXXXXXXXXXXX]
			`,
			`no digest for this image had registry "docker.example.org"`,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("", func(t *testing.T) {
			_, gotErr := selectDigest(tc.dockerInspectOutput, tc.configuredDockerReg)
			if gotErr == nil {
				t.Fatalf("got nil error; want %q", tc.wantErr)
			}
			got := gotErr.Error()
			want := tc.wantErr
			if !strings.Contains(got, want) {
				t.Errorf("got error %q; want error containing %q", got, want)
			}
		})
	}
}
