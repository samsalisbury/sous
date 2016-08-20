package test_with_docker

import (
	"log"
	"testing"
	"time"

	"github.com/nyarly/testify/assert"
)

func TestUntilReady(t *testing.T) {
	assert := assert.New(t)
	log.SetFlags(log.Flags() | log.Lshortfile)

	err := UntilReady(time.Second/10, time.Second, func() (string, func() bool, func()) {
		return "returns quickly",
			func() bool { return true },
			func() {}
	})
	assert.NoError(err)

	err = UntilReady(time.Second/10, time.Second, func() (string, func() bool, func()) {
		return "never returns",
			func() bool { return false },
			func() {}
	})
	assert.Error(err)
}
