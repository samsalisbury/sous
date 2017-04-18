package docker_registry

import (
	"regexp"

	"github.com/opentable/sous/util/spies"
	"github.com/stretchr/testify/mock"
)

type (
	matcher struct {
		*regexp.Regexp
		value interface{}
	}

	valList []interface{}

	call struct {
		method string
		args   valList
		res    valList
	}

	// DummyRegistryClient is a type for use in testing - it supports the Client
	// interface, while only returning metadata that are fed to it
	DummyRegistryClient struct {
		*spies.Spy
	}
)

// NewDummyClient builds and returns a DummyRegistryClient
func NewDummyClient() *DummyRegistryClient {
	return &DummyRegistryClient{
		Spy: spies.NewSpy(),
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

// Cancel fulfills part of Client
func (drc *DummyRegistryClient) Cancel() {}

// BecomeFoolishlyTrusting fulfills part of Client
func (drc *DummyRegistryClient) BecomeFoolishlyTrusting() {}

// GetImageMetadata fulfills part of Client
func (drc *DummyRegistryClient) GetImageMetadata(in, et string) (md Metadata, err error) {
	res := drc.Called(in, et)
	return res.Get(0).(Metadata), res.Error(1)
}

// AllTags fulfills part of Client
func (drc *DummyRegistryClient) AllTags(rn string) (tags []string, err error) {
	res := drc.Called(rn)
	return res.Get(0).([]string), res.Error(1)
}

// LabelsForImageName fulfills part of Client
func (drc *DummyRegistryClient) LabelsForImageName(in string) (labels map[string]string, err error) {
	res := drc.Called(in)
	return res.Get(0).(map[string]string), res.Error(1)
}

// AddMetadata controls the DummyRegistryClient
func (drc *DummyRegistryClient) AddMetadata(pattern string, md Metadata) {
	re := regexp.MustCompile(pattern)
	drc.MatchMethod("GetImageMetadata", func(args mock.Arguments) bool {
		return re.MatchString(args.String(0))
	}, md, nil)
}

// AddTag controls the DummyRegistryClient
func (drc *DummyRegistryClient) AddTag(pattern string, tag []string) {
	re := regexp.MustCompile(pattern)
	drc.MatchMethod("AllTags", func(args mock.Arguments) bool {
		return re.MatchString(args.String(0))
	}, tag, nil)
}

// FeedMetadata is the strings on the marrionette of DummyRegistryClient -
// having triggered a call to GetImageMetadata or LabelsForImageName, use
// FeedMetadata to send the Metadata that the notional docker
// registry might return
func (drc *DummyRegistryClient) FeedMetadata(md Metadata) {
	drc.AddMetadata(`.*`, md)
}

// FeedTags is the strings on the marrionette of DummyRegistryClient -
// having triggered a call to GetImageMetadata or LabelsForImageName, use
// FeedMetadata to send the Metadata that the notional docker
// registry might return
func (drc *DummyRegistryClient) FeedTags(ts []string) {
	drc.AddTag(`.*`, ts)
}
