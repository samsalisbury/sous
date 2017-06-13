package sous

// DummyStateManager is used for testing
type DummyStateManager struct {
	*State
	ReadCount, WriteCount int
}

// ReadState implements StateManager
func (sm *DummyStateManager) ReadState(StateContext) (*State, error) {
	sm.ReadCount++
	return sm.State, nil
}

// WriteState implements StateManager
func (sm *DummyStateManager) WriteState(s *State, c StateContext) error {
	sm.WriteCount++
	*sm.State = *s
	return nil
}
