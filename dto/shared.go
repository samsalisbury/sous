package dto

import sous "github.com/opentable/sous/lib"

type R11nResponse struct {
	QueuePosition int
	// Pointer here is just to allow nil which is a clearer indication of
	// "nothing to see here" than a JSON-marshalled zero value would be.
	Resolution *sous.DiffResolution
}
