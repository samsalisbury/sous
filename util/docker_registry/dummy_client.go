package docker_registry

import (
	"fmt"
	"regexp"

	"github.com/pkg/errors"
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

func (c call) results(res ...interface{}) []interface{} {
	c.res = res
	return res
}

// Cancel fulfills part of Client
func (drc *DummyRegistryClient) Cancel() {}

// BecomeFoolishlyTrusting fulfills part of Client
func (drc *DummyRegistryClient) BecomeFoolishlyTrusting() {}

func (m matcher) String() string {
	return fmt.Sprintf("%v: %v", m.Regexp, m.value)
}

func (drc *DummyRegistryClient) String() string {
	out := "Metadata:\n"
	for _, md := range drc.mds {
		out = out + fmt.Sprintf("%v\n", md)
	}
	out = out + "\nTags:\n"

	for _, t := range drc.ts {
		out = out + fmt.Sprintf("%v\n", t)
	}
	out = out + "\nCalls:\n"

	for _, c := range drc.calls {
		out = out + fmt.Sprintf("%v\n", c)
	}
	return out
}

// GetImageMetadata fulfills part of Client
func (drc *DummyRegistryClient) GetImageMetadata(in, et string) (md Metadata, err error) {
	call := call{method: "GetImageMetadata", args: valList{in, et}}
	defer func() {
		call.res = valList{md, err}
		drc.calls = append(drc.calls, call)
	}()

	m := findMatch(in, drc.mds)
	if m == nil {
		err = errors.Errorf("No match for %q", in)
		return
	}
	md = m.value.(Metadata)
	return
}

// AllTags fulfills part of Client
func (drc *DummyRegistryClient) AllTags(rn string) (tags []string, err error) {
	call := call{method: "AllTags", args: valList{rn}}
	defer func() {
		call.res = valList{tags, err}
		drc.calls = append(drc.calls, call)
	}()

	m := findMatch(rn, drc.ts)
	if m == nil {
		return
	}
	tags = m.value.([]string)
	return
}

// LabelsForImageName fulfills part of Client
func (drc *DummyRegistryClient) LabelsForImageName(in string) (labels map[string]string, err error) {
	call := call{method: "LabelsForImageName", args: valList{in}}
	defer func() {
		call.res = valList{labels, err}
		drc.calls = append(drc.calls, call)
	}()

	md, err := drc.GetImageMetadata(in, "")
	if err != nil {
		return
	}

	labels = md.Labels
	return
}

// CallsTo returns a filtered list of the calls to this spy: those that were made to the named method
func (drc *DummyRegistryClient) CallsTo(name string) []call {
	calls := []call{}
	for _, c := range drc.calls {
		if c.method == name {
			calls = append(calls, c)
		}
	}
	return calls
}

// AddMetadata controls the DummyRegistryClient
func (drc *DummyRegistryClient) AddMetadata(pattern string, md Metadata) {
	drc.mds = append(drc.mds, matcher{Regexp: regexp.MustCompile(pattern), value: md})
}

// AddTag controls the DummyRegistryClient
func (drc *DummyRegistryClient) AddTag(pattern string, tag []string) {
	drc.ts = append(drc.ts, matcher{Regexp: regexp.MustCompile(pattern), value: tag})
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
