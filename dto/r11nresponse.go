package dto

import sous "github.com/opentable/sous/lib"

//R11nResponse dto used by server to return single deploy status, read by client
type R11nResponse struct {
	QueuePosition int
	// Pointer here is just to allow nil which is a clearer indication of
	// "nothing to see here" than a JSON-marshalled zero value would be.
	Resolution *sous.DiffResolution
}
