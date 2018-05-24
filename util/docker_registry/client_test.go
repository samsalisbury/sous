package docker_registry

import (
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

func TestRegistries(t *testing.T) {
	assert := assert.New(t)

	rs := NewRegistries()
	r := &registry{}
	assert.NoError(rs.AddRegistry("x", r))
	assert.Equal(rs.GetRegistry("x"), r)
	assert.NoError(rs.DeleteRegistry("x"))
	assert.Nil(rs.GetRegistry("x"))
}

// This test is terrible, but the current design of the client is hard to test
func TestNewClient(t *testing.T) {
	assert := assert.New(t)

	c := NewClient(logging.SilentLogSet())
	assert.NotNil(c)
	c.Cancel()
}

func TestSplitHost(t *testing.T) {
	url, ref, err := splitHost("192.168.11.1:5000/some/repo:some-tag")
	wantURL := "192.168.11.1:5000"
	if url != wantURL {
		t.Fatalf("got url=%q; want %q", url, wantURL)
	}
	wantRef := "some/repo:some-tag"
	if ref.String() != wantRef {
		t.Fatalf("got ref=%q; want %q", ref, wantRef)
	}
	if err != nil {
		t.Fatalf("got error %q; want nil", err)
	}
}
