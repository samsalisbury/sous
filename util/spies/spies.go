package spies

import (
	"regexp"

	"github.com/stretchr/testify/mock"
)

type (
	matcher struct {
		*regexp.Regexp
		value interface{}
	}

	call struct {
		method string
		args   mock.Arguments
		res    mock.Arguments
	}

	// DummyRegistryClient is a type for use in testing - it supports the Client
	// interface, while only returning metadata that are fed to it
	DummyRegistryClient struct {
		mds []matcher
		ts  []matcher

		calls []call
	}
)

// NewDummyClient builds and returns a DummyRegistryClient
func NewDummyClient() *DummyRegistryClient {
	return &DummyRegistryClient{
		mds: []matcher{},
		ts:  []matcher{},

		calls: []call{},
	}
}

func findMatch(str string, list []matcher) *matcher {
	for _, m := range list {
		if m.MatchString(str) {
			return &m
		}
	}

	return nil
}

func (c *call) results(res ...interface{}) []interface{} {
	c.res = res
	return res
}
