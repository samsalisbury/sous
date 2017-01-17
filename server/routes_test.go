package server

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
)

func testInject(thing interface{}) error {
	processGraph := graph.BuildTestGraph(&bytes.Buffer{}, os.Stdout, os.Stdout)
	processGraph.Add(&config.Verbosity{})

	requestGraph := BuildRequestGraph(processGraph)

	mh := &MetaHandler{processGraph: processGraph, requestGraph: requestGraph}
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		panic(err)
	}
	p := httprouter.Params{}

	processGraph.MustInject(thing)

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
