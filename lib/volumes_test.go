package sous

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type vpair struct {
	v Volumes
	i int
}

func TestVolumesEqual(t *testing.T) {
	assert := assert.New(t)
	vs := []vpair{
		vpair{Volumes{&Volume{"a", "a", "RO"}, &Volume{"a", "a", "RO"}}, 1},
		vpair{Volumes{&Volume{"a", "a", "RO"}, &Volume{"a", "a", "RO"}}, 1},
		vpair{Volumes{&Volume{"a", "a", "RO"}}, 4},
		vpair{Volumes{&Volume{"a", "b", "RO"}, &Volume{"a", "a", "RO"}}, 2},
		vpair{Volumes{&Volume{"a", "a", "RW"}, &Volume{"a", "a", "RO"}}, 3},
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
