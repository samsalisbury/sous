package singularity

import "sync/atomic"

var deployTimestampCounter int64 = 1

func nextDeployTimestamp() int64 {
	return atomic.AddInt64(&deployTimestampCounter, 1)
}
