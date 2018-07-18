package server

import (
	"testing"

	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/assert"
)

func TestHandleDefault_Get(t *testing.T) {
	rm := restful.BuildRouteMap(func(re restful.RouteEntryBuilder) {
		re("test", "/test", nil)
	})
	h := &getDefaultHandler{
		routeMap: *rm,
	}
	expected := Default{
		Paths: []string{"/test"},
	}

	data, statusCode := h.Exchange()
	assert.Equal(t, 200, statusCode)
	assert.Equal(t, expected, data)
}
