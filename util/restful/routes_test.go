package restful

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/nyarly/testify/assert"
	"github.com/samsalisbury/psyringe"
)

type testInjectedHandler struct {
	*ResponseWriter
	*http.Request
	httprouter.Params
	*QueryValues
}

func testInject(thing interface{}) error {
	gf := func() Injector {
		return psyringe.New()
	}
	mh := &MetaHandler{graphFac: gf}
	w := httptest.NewRecorder()
	r, err := http.NewRequest("GET", "/test", nil)
	if err != nil {
		panic(err)
	}
	p := httprouter.Params{}

	return mh.ExchangeGraph(w, r, p).Inject(thing)
}

func TestInjectsArtifactHandler(t *testing.T) {
	th := &testInjectedHandler{}
	testInject(th)
	assert.NotNil(t, th.ResponseWriter)
	assert.NotNil(t, th.Request)
	assert.NotNil(t, th.Params)
	assert.NotNil(t, th.QueryValues)
}
