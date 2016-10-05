package sous

type HTTPStateManager struct {
}

func (hsm *HTTPStateManager) ReadState() (*State, error) {}

func (hsm *HTTPStateManager) WriteState(*State) error {}
