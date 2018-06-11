package sous

import (
	"testing"

	"github.com/nyarly/spies"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/opentable/sous/util/restful/restfultest"
	"github.com/stretchr/testify/assert"
)

type dispatchSMScenario struct {
	dsm *DispatchStateManager

	httpClients  map[string]*spies.Spy
	httpUpdaters map[string]*spies.Spy
	local        StateManagerController
}

func setupDispatchStateManager(t *testing.T) dispatchSMScenario {
	state := DefaultStateFixture()
	deployments, err := state.Deployments()
	if err != nil {
		t.Fatal(err)
	}

	localCluster := "local"
	clusters := []string{"cluster1", "cluster2"}

	local, lc := NewStateManagerSpy()
	lc.MatchMethod("ReadState", spies.AnyArgs, state, nil)
	lc.MatchMethod("ReadCluster", spies.AnyArgs, deployments, nil)

	lup, lupc := restfultest.NewUpdateSpy()
	cluster1, rlc := restfultest.NewHTTPClientSpy()
	rlc.MatchMethod("Retrieve", spies.AnyArgs, state, lup, nil)
	rlc.MatchMethod("ReadCluster", spies.AnyArgs, deployments, nil)

	rup, rupc := restfultest.NewUpdateSpy()
	cluster2, rrc := restfultest.NewHTTPClientSpy()
	rrc.MatchMethod("Retrieve", spies.AnyArgs, state, rup, nil)
	rrc.MatchMethod("ReadCluster", spies.AnyArgs, deployments, nil)

	wup, wupc := restfultest.NewUpdateSpy()
	whole, rwc := restfultest.NewHTTPClientSpy()
	rwc.MatchMethod("Retrieve", spies.AnyArgs, state, wup, nil)
	rwc.MatchMethod("ReadCluster", spies.AnyArgs, deployments, nil)

	tid := TraceID("testtrace")

	ls, _ := logging.NewLogSinkSpy()

	remote := NewHTTPStateManager(whole, tid, ls)
	remote.clusterClients = map[string]restful.HTTPClient{}
	remote.clusterClients["cluster1"] = cluster1
	remote.clusterClients["cluster2"] = cluster2

	dsm := NewDispatchStateManager(localCluster, clusters, local, remote, ls)

	return dispatchSMScenario{
		dsm: dsm,
		httpClients: map[string]*spies.Spy{
			"whole":    rwc,
			"cluster1": rlc,
			"cluster2": rrc,
		},
		httpUpdaters: map[string]*spies.Spy{
			"whole":    wupc,
			"cluster1": lupc,
			"cluster2": rupc,
		},
		local: lc,
	}
}

func TestDispatchStateManagerRead(t *testing.T) {
	scenario := setupDispatchStateManager(t)

	state, err := scenario.dsm.ReadState()

	assert.NoError(t, err)
	assert.NotNil(t, state)

	assert.Len(t, scenario.httpClients["whole"].CallsTo("Retrieve"), 0)
	if assert.Len(t, scenario.httpClients["cluster1"].CallsTo("Retrieve"), 1) {
		retrieve := scenario.httpClients["cluster1"].CallsTo("Retrieve")[0]
		assert.Equal(t, retrieve.PassedArgs().String(0), "./state/deployments")
	}
	if assert.Len(t, scenario.httpClients["cluster2"].CallsTo("Retrieve"), 1) {
		retrieve := scenario.httpClients["cluster2"].CallsTo("Retrieve")[0]
		assert.Equal(t, retrieve.PassedArgs().String(0), "./state/deployments")
	}
	assert.Len(t, scenario.local.CallsTo("ReadState"), 2)
}

func TestDispatchStateManagerWrite(t *testing.T) {
	scenario := setupDispatchStateManager(t)

	s := NewState()
	u := User{}

	err := scenario.dsm.WriteState(s, u)

	assert.NoError(t, err)

	assert.Len(t, scenario.httpClients["whole"].CallsTo("Retrieve"), 0)
	assert.Len(t, scenario.httpUpdaters["whole"].CallsTo("Update"), 0)

	if assert.Len(t, scenario.httpClients["cluster1"].CallsTo("Retrieve"), 1) {
		retrieve := scenario.httpClients["cluster1"].CallsTo("Retrieve")[0]
		assert.Equal(t, retrieve.PassedArgs().String(0), "./state/deployments")
	}
	assert.Len(t, scenario.httpUpdaters["cluster1"].CallsTo("Update"), 1)

	if assert.Len(t, scenario.httpClients["cluster2"].CallsTo("Retrieve"), 1) {
		retrieve := scenario.httpClients["cluster2"].CallsTo("Retrieve")[0]
		assert.Equal(t, retrieve.PassedArgs().String(0), "./state/deployments")
	}
	assert.Len(t, scenario.httpUpdaters["cluster2"].CallsTo("Update"), 1)

	assert.Len(t, scenario.local.CallsTo("ReadState"), 1)
	assert.Len(t, scenario.local.CallsTo("WriteState"), 1)
}
