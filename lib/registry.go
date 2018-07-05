package sous

import (
	"github.com/nyarly/spies"
)

type (
	// ImageLabeller can get the image labels for a given imageName
	ImageLabeller interface {
		//ImageLabels finds the (docker) labels for a given image name
		ImageLabels(imageName string) (labels map[string]string, err error)
	}

	// Registry describes a system for mapping SourceIDs to BuildArtifacts and vice versa
	Registry interface {
		ImageLabeller
		// GetArtifact gets the build artifact address for a source ID.
		// It does not guarantee that that artifact exists.
		GetArtifact(SourceID) (*BuildArtifact, error)
		// GetSourceID gets the source ID associated with the
		// artifact, regardless of the existence of the artifact.
		GetSourceID(*BuildArtifact) (SourceID, error)
		// GetMetadata returns metadata for a source ID.
		//GetMetadata(SourceID) (map[string]string, error)

		// ListSourceIDs returns a list of known SourceIDs
		ListSourceIDs() ([]SourceID, error)

		// Warmup requests that the registry check specific artifact names for existence
		// the details of this behavior will vary by implementation. For Docker, for instance,
		// the corresponding repo is enumerated
		Warmup(string) error
	}

	// An Inserter puts data into a registry.
	Inserter interface {
		// Insert pairs a SourceID with a build artifact.
		Insert(sid SourceID, ba BuildArtifact) error
	}

	// An InserterSpy is a spy implementation of the Inserter interface
	InserterSpy struct {
		*spies.Spy
	}

	// A RegistrySpy is a spy implementation of the Registry interface
	RegistrySpy struct {
		spy *spies.Spy
	}

	// ClientInsert, Client version of Inserter, differenciates between Server for graph
	ClientInserter struct{ Inserter }
)

// NewInserterSpy returns a spy inserter for testing
func NewInserterSpy() (InserterSpy, *spies.Spy) {
	ctrl := spies.NewSpy()
	return InserterSpy{ctrl}, ctrl
}

// Insert implements Inserter on InserterSpy
func (is InserterSpy) Insert(sid SourceID, ba BuildArtifact) error {
	return is.Called(sid, ba).Error(0)
}

// NewRegistrySpy returns a spy Registry for testing.
func NewRegistrySpy() (RegistrySpy, *spies.Spy) {
	spy := spies.NewSpy()
	return RegistrySpy{spy: spy}, spy
}

// ImageLabels implements Registry on RegistrySpy.
func (spy RegistrySpy) ImageLabels(imageName string) (labels map[string]string, err error) {
	res := spy.spy.Called(imageName)
	return res.Get(0).(map[string]string), res.Error(1)
}

// GetArtifact implements Registry on RegistrySpy.
func (spy RegistrySpy) GetArtifact(sid SourceID) (*BuildArtifact, error) {
	res := spy.spy.Called(sid)
	return res.Get(0).(*BuildArtifact), res.Error(1)
}

// GetSourceID implements Registry on RegistrySpy.
func (spy RegistrySpy) GetSourceID(art *BuildArtifact) (SourceID, error) {
	res := spy.spy.Called(art)
	return res.Get(0).(SourceID), res.Error(1)
}

// ListSourceIDs implements Registry on RegistrySpy.
func (spy RegistrySpy) ListSourceIDs() ([]SourceID, error) {
	res := spy.spy.Called()
	return res.Get(0).([]SourceID), res.Error(1)
}

// Warmup implements Registry on RegistrySpy.
func (spy RegistrySpy) Warmup(name string) error {
	res := spy.spy.Called(name)
	return res.Error(0)
}
