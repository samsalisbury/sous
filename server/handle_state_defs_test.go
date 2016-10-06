package server

import (
	"testing"

	"github.com/opentable/sous/lib"
)

func TestStateDefGet(t *testing.T) {
	th := &StateDefGetHandler{
		State: &sous.State{
			Defs: sous.Defs{
				DockerRepo: "reponame",
			},
		},
	}

	data, status := th.Exchange()
	if status != 200 {
		t.Errorf("Status was %v not 200", status)
	}

	if defs, ok := data.(sous.Defs); ok {
		if defs.DockerRepo != "reponame" {
			t.Errorf("returned Defs didn't include given data")
		}

	} else {
		t.Errorf("returned data wasn't a sous.Defs: %T", defs)
	}
}
