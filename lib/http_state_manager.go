package sous

type HTTPStateManager struct {
	cached *State
}

func (hsm *HTTPStateManager) ReadState() (*State, error) {
	return nil, nil
}

func (hsm *HTTPStateManager) WriteState(*State) error {
	return nil
}
