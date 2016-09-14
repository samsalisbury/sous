package server

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/opentable/sous/config"
)

func TestOverallRouter(t *testing.T) {
	assert := assert.New(t)

	ts := httptest.NewServer(NewSousRouter(&config.Verbosity{}))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/gdm")
	assert.NoError(err)
	gdm, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	assert.NoError(err)
	assert.Regexp(`"Deployments"`, string(gdm))
}
