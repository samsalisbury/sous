package readdebugger

import (
	"io"
)

type (
	readDebugger struct {
		wrapped         io.Reader
		logging, logged bool
		count           int
		read            []byte
		log             func([]byte, int, error)
	}
)

// New creates a new readDebugger that wraps a ReadCloser.
func New(rc io.Reader, log func([]byte, int, error)) io.ReadCloser {
	switch r := rc.(type) {
	default:
		return &readDebugger{
			wrapped: rc,
			read:    []byte{},
			log:     log,
		}
	case *readDebugger:
		return r
	}
}

// Read implements Reader on readDebugger.
func (rd *readDebugger) Read(p []byte) (int, error) {
	n, err := rd.wrapped.Read(p)
	rd.read = append(rd.read, p[:n]...)
	rd.count += n
	if !rd.logging && err != nil {
		rd.logging = true
		defer func() { rd.logging = false }()
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
	if !rd.logging && !rd.logged {
		rd.logging = true
		defer func() { rd.logging = false }()
		rd.logged = true
		rd.log(rd.read, rd.count, err)
	}
	return err
}
