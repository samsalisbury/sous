package server

import (
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/opentable/sous/config"
)

func TestHandleServerList_Get(t *testing.T) {
	assert := assert.New(t)

	h := &ServerListHandler{
		Config: &config.Config{
			SiblingURLs: []string{"https://left.sous.com", "https://right.sous.com"},
		},
	}

	rez, stat := h.Exchange()
	assert.Equal(stat, 200)

	list, yup := rez.(serverListData)
	assert.True(yup)
	assert.Equal(list.Servers[0].URL, "https://left.sous.com")
	assert.Equal(list.Servers[1].URL, "https://right.sous.com")
}
