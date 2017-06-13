package sous

// StateContext contains additional data about what is being read or written
// by a StateManager.
type StateContext struct {
	// User is the user this write is attributed to.
	User User
	// TargetManifestID is the manifest this write is expected to affect.
	// Implementations of StateWriter.WriteState should check that the
	// change being written corresponds with this manifest ID.
	TargetManifestID ManifestID
}
