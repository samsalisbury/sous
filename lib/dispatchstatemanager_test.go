package sous

import (
	"testing"

	"github.com/nyarly/spies"
	"github.com/opentable/sous/util/logging"
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
	state := NewState()

	localCluster := "local"
	clusters := []string{"left", "right"}

	local, lc := NewStateManagerSpy()
	lc.MatchMethod("ReadState", spies.AnyArgs, state, nil)

	/*
		lup, lupc := restfultest.NewUpdateSpy()
		left, rlc := restfultest.NewHTTPClientSpy()
		rlc.MatchMethod("Retrieve", spies.AnyArgs, state, lup, nil)

		rup, rupc := restfultest.NewUpdateSpy()
		right, rrc := restfultest.NewHTTPClientSpy()
		rrc.MatchMethod("Retrieve", spies.AnyArgs, state, rup, nil)
	*/

	wup, wupc := restfultest.NewUpdateSpy()
	whole, rwc := restfultest.NewHTTPClientSpy()
	rwc.MatchMethod("Retrieve", spies.AnyArgs, state, wup, nil)

	tid := TraceID("testtrace")

	ls, _ := logging.NewLogSinkSpy()

	remote := NewHTTPStateManager(whole, tid, ls)

	dsm := NewDispatchStateManager(localCluster, clusters, local, remote, ls)

	return dispatchSMScenario{
		dsm: dsm,
		httpClients: map[string]*spies.Spy{
			"whole": rwc,
			//"left":  rlc,
			//"right": rrc,
		},
		httpUpdaters: map[string]*spies.Spy{
			"whole": wupc,
			//"left":  lupc,
			//"right": rupc,
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
	if assert.Len(t, scenario.httpClients["left"].CallsTo("Retrieve"), 1) {
		retrieve := scenario.httpClients["left"].CallsTo("Retrieve")[0]
		assert.Equal(t, retrieve.PassedArgs().String(0), "./state/deployments")
	}
	if assert.Len(t, scenario.httpClients["right"].CallsTo("Retrieve"), 1) {
		retrieve := scenario.httpClients["right"].CallsTo("Retrieve")[0]
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

	if assert.Len(t, scenario.httpClients["left"].CallsTo("Retrieve"), 1) {
		retrieve := scenario.httpClients["left"].CallsTo("Retrieve")[0]
		assert.Equal(t, retrieve.PassedArgs().String(0), "./state/deployments")
	}
	assert.Len(t, scenario.httpUpdaters["left"].CallsTo("Update"), 1)

	if assert.Len(t, scenario.httpClients["right"].CallsTo("Retrieve"), 1) {
		retrieve := scenario.httpClients["right"].CallsTo("Retrieve")[0]
		assert.Equal(t, retrieve.PassedArgs().String(0), "./state/deployments")
	}
	assert.Len(t, scenario.httpUpdaters["right"].CallsTo("Update"), 1)

	assert.Len(t, scenario.local.CallsTo("ReadState"), 1)
	assert.Len(t, scenario.local.CallsTo("WriteState"), 1)
}
