package sous

import "io"

type (
	// ReadDebugger wraps a ReadCloser and logs the data as it buffers past.
	ReadDebugger struct {
		wrapped io.ReadCloser
		logged  bool
		count   int
		read    []byte
		log     func([]byte, int, error)
	}
)

// NewReadDebugger creates a new ReadDebugger that wraps a ReadCloser.
func NewReadDebugger(rc io.ReadCloser, log func([]byte, int, error)) *ReadDebugger {
	return &ReadDebugger{
		wrapped: rc,
		read:    []byte{},
		log:     log,
	}
}

// Read implements Reader on ReadDebugger.
func (rd *ReadDebugger) Read(p []byte) (int, error) {
	n, err := rd.wrapped.Read(p)
	rd.read = append(rd.read, p[:n]...)
	rd.count += n
	if err != nil {
		rd.log(rd.read, rd.count, err)
		rd.logged = true
	}
	return n, err
}

// Close implements Closer on ReadDebugger.
func (rd *ReadDebugger) Close() error {
	err := rd.wrapped.Close()
	if !rd.logged {
		rd.log(rd.read, rd.count, err)
		rd.logged = true
	}
	return err
}
