package server

import (
	"net/http/httptest"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/assert"
)

func TestHandlesGDMGet(t *testing.T) {
	assert := assert.New(t)

	w := httptest.NewRecorder()
	//etag := "swordfish"

	th := &GETGDMHandler{
		RzWriter: &restful.ResponseWriter{w},
		LogSet:   &logging.Log,
		GDM:      sous.NewState(),
	}

	data, status := th.Exchange()
	//assert.Equal(w.Header().Get("Etag"), etag)
	assert.Equal(status, 200)
	assert.Len(data.(GDMWrapper).Deployments, 0)
}
