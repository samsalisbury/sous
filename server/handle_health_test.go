package server

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

func TestHandleServerList_Get(t *testing.T) {

	h := &getHealthHandler{}
	rez, stat := h.Exchange()

	if stat != 200 {
		t.Errorf("Expecting 200 status; got %d", stat)
	}

	if rez.Version != sous.Version {
		t.Errorf("Expecting %q; got %q", sous.Version, rez.Version)
	}
}
