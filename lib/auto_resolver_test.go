package sous

import (
	"testing"
	"time"

	"github.com/nyarly/testify/assert"
)

func dummyResolver() *Resolver {
	return NewResolver(NewDummyDeployer(), NewDummyRegistry())
}

func setupAR() *AutoResolver {
	return &AutoResolver{
		UpdateTime:  30 * time.Second,
		Resolver:    dummyResolver(),
		StateReader: &DummyStateManager{},
	}
}

func TestDone(t *testing.T) {
	assert := assert.New(t)

	ar := setupAR()

	done := ar.kickoff()
	assert.NotPanics(func() { close(done) }) //pretty crap test, honestly
}
