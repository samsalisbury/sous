package git

import (
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParseTags(t *testing.T) {
	assert := assert.New(t)

	lines := []string{
		"2f381f35ffcf57b21b4c2991635f7f825c29f003 2016-08-02T11:16:04-04:00 HEAD -> master, tag: 0.2.0, origin/master, origin/HEAD",
		"0a271b999974db8f37326e16c9027436098b251f 2016-08-01T10:54:53-04:00 tag: 0.1.6",
		"a83cebbdb1e06bc325f88d953120ac60dead7268 2016-07-26T15:32:24-04:00 tag: 0.1.5",
		"65017e8bcbeaf10d48ea677113a3d5d99ed03c45 2016-07-12T14:23:03-04:00 tag: 0.1.4",
		"6013d74276ac6a62f8fec3aee7c06162496b25a5 2016-07-12T13:14:52-04:00 tag: 0.1.3",
		"b779b60a13f0a3b8edbecf81b0b24b252202b3a2 2016-07-11T16:00:14-04:00 tag: 0.1.2",
		"612eec227ccdcdbe9d182ce830adb566930ca0c0 2016-07-07T14:41:18-04:00 2f381f35ffcf57b21b4c2991635f7f825c29f003",
	}

	tags := (&Client{}).parseTags(lines)
	assert.Len(tags, 6)
	assert.Contains(tags, sous.Tag{Name: "0.2.0", Revision: "2f381f35ffcf57b21b4c2991635f7f825c29f003"})
}

func TestNewClient(t *testing.T) {
	c, err := NewClient(&shell.Sh{})
	if err != nil {
		t.Fatal(err)
	}
	switch interface{}(c).(type) {
	case *Client:
		t.Logf("NewClient() created a *Client %v\n", c)
	default:
		t.Fatalf("NewClient() did not create a *Client")
	}
}
