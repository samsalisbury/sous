package sous

import (
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

type vpair struct {
	v Volumes
	i int
}

func TestVolumesEqual(t *testing.T) {
	assert := assert.New(t)
	log, _ := logging.NewLogSinkSpy()
	vs := []vpair{
		vpair{Volumes{&Volume{"a", "a", "RO", &log}, &Volume{"a", "a", "RO", &log}}, 1},
		vpair{Volumes{&Volume{"a", "a", "RO", &log}, &Volume{"a", "a", "RO", &log}}, 1},
		vpair{Volumes{&Volume{"a", "a", "RO", &log}}, 4},
		vpair{Volumes{&Volume{"a", "b", "RO", &log}, &Volume{"a", "a", "RO", &log}}, 2},
		vpair{Volumes{&Volume{"a", "a", "RW", &log}, &Volume{"a", "a", "RO", &log}}, 3},
	}

	for _, l := range vs {
		for _, r := range vs {
			if l.i == r.i {
				assert.True(l.v.Equal(r.v))
			} else {
				assert.False(l.v.Equal(r.v))
			}
		}
	}
}
