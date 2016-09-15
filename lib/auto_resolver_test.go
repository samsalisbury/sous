package sous

import (
	"io/ioutil"
	"log"
	"testing"
	"time"

	"github.com/nyarly/testify/assert"
)

func dummyResolver() *Resolver {
	registry := NewDummyRegistry()
	drc := NewDummyRectificationClient(registry)
	drc.SetLogger(log.New(ioutil.Discard, "rectify: ", 0))
	return NewResolver(drc, registry)
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
	assert.NotPanics(close(done)) //pretty crap test, honestly
}
