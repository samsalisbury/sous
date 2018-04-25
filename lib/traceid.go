package sous

import "github.com/opentable/sous/util/logging"

// A TraceID is the header to add to requests for tracing purposes.
type TraceID string

// EachField implements EachFielder on TraceID
func (tid TraceID) EachField(fn logging.FieldReportFn) {
	fn(logging.RequestId, tid)
}
