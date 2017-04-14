package server

import (
	"testing"

	"github.com/opentable/sous/config"
	"github.com/stretchr/testify/assert"
)

func TestHandleServerList_Get(t *testing.T) {
	assert := assert.New(t)

	h := &ServerListHandler{
		Config: &config.Config{
			SiblingURLs: map[string]string{"left": "https://left.sous.com", "right": "https://right.sous.com"},
		},
	}

	rez, stat := h.Exchange()
	assert.Equal(stat, 200)

	list, yup := rez.(serverListData)
	assert.True(yup)

	if list.Servers[0].ClusterName == "right" {
		list.Servers = []server{list.Servers[1], list.Servers[0]}
	}

	// test predates config []string -> map[string]string
	assert.Equal(list.Servers[0].URL, "https://left.sous.com")
	assert.Equal(list.Servers[1].URL, "https://right.sous.com")

	// newer test
	assert.Equal(list.Servers[0].ClusterName, "left")
	assert.Equal(list.Servers[1].ClusterName, "right")
}
