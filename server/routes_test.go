package server

import (
	"log"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
)

func testInject(thing interface{}) error {
	gf := func() Injector {
		g := graph.BuildGraph(os.Stdout, os.Stdout)
		g.Add(&config.Verbosity{})
		return g
	}
	mh := &MetaHandler{graphFac: gf}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/test", nil)
	p := httprouter.Params{}

	return mh.ExchangeGraph(w, r, p).Inject(thing)
}

func TestInjectsArtifactHandler(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	pah := &PUTArtifactHandler{}
	testInject(pah)
	if pah.Inserter == nil {
		t.Errorf("pah.Inserter nil: %#v", pah)
	}
	if pah.QueryValues == nil {
		t.Errorf("pah.QueryValues nil: %#v", pah)
	}
	if pah.Request == nil {
		t.Errorf("pah.Request nil: %#v", pah)
	}
}
