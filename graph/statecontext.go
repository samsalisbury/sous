package graph

import sous "github.com/opentable/sous/lib"

// StateWriteContext returns a sous.StateContext configured for writing.
type StateWriteContext sous.StateWriteContext

// StateReadContext returns a sous.StateContext configured for reading.
type StateReadContext sous.StateWriteContext

func newStateWriteContext(mid TargetManifestID, u sous.User) StateWriteContext {
	return StateWriteContext{
		User:             u,
		TargetManifestID: sous.ManifestID(mid),
	}
}

func newStateReadContext(u sous.User) StateWriteContext {
	return StateWriteContext{
		User: u,
	}
}
