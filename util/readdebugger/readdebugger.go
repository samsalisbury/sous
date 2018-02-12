package readdebugger

import (
	"io"
)

type (
	readDebugger struct {
		wrapped io.Reader
		logged  bool
		count   int
		read    []byte
		log     func([]byte, int, error)
	}
)

// New creates a new readDebugger that wraps a ReadCloser.
func New(rc io.Reader, log func([]byte, int, error)) io.ReadCloser {
	return &readDebugger{
		wrapped: rc,
		read:    []byte{},
		log:     log,
	}
}

// Read implements Reader on readDebugger.
func (rd *readDebugger) Read(p []byte) (int, error) {
	n, err := rd.wrapped.Read(p)
	rd.read = append(rd.read, p[:n]...)
	rd.count += n
	if err != nil {
		rd.log(rd.read, rd.count, err)
		rd.logged = true
	}
	return n, err
}

// Close implements Closer on readDebugger.
func (rd *readDebugger) Close() (err error) {
	if cl, is := rd.wrapped.(io.Closer); is {
		err = cl.Close()
	}
	if !rd.logged {
		rd.log(rd.read, rd.count, err)
		rd.logged = true
	}
	return err
}
