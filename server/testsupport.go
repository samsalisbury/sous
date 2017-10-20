package server

import (
	"net/http"
	"os"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/samsalisbury/semv"
)

func TestingInMemoryClient() (restful.HTTPClient, error) {
	inserter := sous.NewInserterSpy()

	state := sous.NewState()
	state.Defs = sous.Defs{
		DockerRepo: "",
		Clusters: map[string]*sous.Cluster{
			"test-cluster": {
				Name:    "test-cluster",
				Kind:    "",
				BaseURL: "",
				Env: map[string]sous.Var{
					"X": "1",
				},
				Startup: sous.Startup{
					SkipCheck:                 true,
					ConnectDelay:              0,
					Timeout:                   0,
					ConnectInterval:           0,
					CheckReadyProtocol:        "",
					CheckReadyURIPath:         "",
					CheckReadyPortIndex:       0,
					CheckReadyFailureStatuses: nil,
					CheckReadyURITimeout:      0,
					CheckReadyInterval:        0,
					CheckReadyRetries:         0,
				},
				AllowedAdvisories: []string{},
			},
		},
		EnvVars:   sous.EnvDefs{},
		Resources: sous.FieldDefinitions{},
		Metadata:  sous.FieldDefinitions{},
	}

	ls := logging.NewLogSet(semv.MustParse("1.1.1"), "", "", os.Stderr)

	locator := ComponentLocator{
		LogSink:       ls,
		Config:        &config.Config{},
		Inserter:      inserter,
		StateManager:  &sous.DummyStateManager{State: state},
		ResolveFilter: &sous.ResolveFilter{},
		AutoResolver:  &sous.AutoResolver{},
	}

	handler := Handler(locator, http.NotFoundHandler(), ls)

	return restful.NewInMemoryClient(handler, ls, map[string]string{"X-Gatelatch": os.Getenv("GATELATCH")})
}
