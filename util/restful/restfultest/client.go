package restfultest

import (
	"encoding/json"

	"github.com/nyarly/spies"
	"github.com/opentable/sous/util/restful"
)

type (
	// HTTPClientSpy is a spy implementation of restful.HTTPClient
	HTTPClientSpy struct {
		*spies.Spy
	}

	// UpdateSpy is a spy implementation of restful.Updater
	UpdateSpy struct {
		*spies.Spy
	}
	/*
		// HTTPClient interacts with a HTTPServer
		//   It's designed to handle basic CRUD operations in a safe and restful way.
		HTTPClient interface {
			Create(urlPath string, qParms map[string]string, rqBody interface{}, headers map[string]string) error
			Retrieve(urlPath string, qParms map[string]string, rzBody interface{}, headers map[string]string) (Updater, error)
			Delete(urlPath string, qParms map[string]string, from *resourceState, headers map[string]string) error
		}

		// An Updater captures the state of a retrieved resource so that it can be updated later.
		Updater interface {
			Update(params map[string]string, body Comparable, headers map[string]string) error
		}
	*/
)

func roundtrip(in, out interface{}) {
	bs, err := json.Marshal(in)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(bs, out); err != nil {
		panic(err)
	}
}

// NewHTTPClientSpy creates a new spy implementation of restful.HTTPClient
// It returns a restful.HTTPClient and the spy manager
func NewHTTPClientSpy() (restful.HTTPClient, *spies.Spy) {
	spy := spies.NewSpy()
	return &HTTPClientSpy{spy}, spy
}

// NewUpdateSpy creates a new spy implementation of restful.Update
// It returns a restful.Update and the spy manager
func NewUpdateSpy() (restful.UpdateDeleter, *spies.Spy) {
	spy := spies.NewSpy()
	return &UpdateSpy{spy}, spy
}

// DummyUpdater returns an unconfigured UpdateSpy. Mostly useful because it's single-result.
func DummyUpdater() restful.UpdateDeleter {
	s, _ := NewUpdateSpy()
	return s
}

// Create is a spy implementation of the restful.HTTPClient.Create method
func (c *HTTPClientSpy) Create(url string, ps map[string]string, bd interface{}, hs map[string]string) (restful.UpdateDeleter, error) {
	res := c.Called(url, ps, bd, hs)
	roundtrip(res.Get(0), bd)
	return res.Get(1).(restful.UpdateDeleter), res.Error(2)
}

// Retrieve is a spy implementation of the restful.HTTPClient.Retrieve method
func (c *HTTPClientSpy) Retrieve(url string, ps map[string]string, bd interface{}, hs map[string]string) (restful.UpdateDeleter, error) {
	res := c.Called(url, ps, bd, hs)
	roundtrip(res.Get(0), bd)

	return res.Get(1).(restful.UpdateDeleter), res.Error(2)
}

// Update is a spy implementation of the restful.UpdateDeleter.Update method
func (u *UpdateSpy) Update(bd restful.Comparable, hs map[string]string) error {
	res := u.Called(bd, hs)
	return res.Error(0)
}

// Delete is a spy implementation of the restful.UpdateDeleter.Delete method
func (u *UpdateSpy) Delete(hs map[string]string) error {
	res := u.Called(hs)
	return res.Error(0)
}

// Location is a spy implemention of restful.UpdateDeleter.Location method
func (u *UpdateSpy) Location() string {
	res := u.Called()
	return res.String(0)
}
