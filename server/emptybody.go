package server

import (
	"io"
	"io/ioutil"
)

type (
	// EmptyReader is an empty reader - returns EOF immediately
	EmptyReader struct{}
)

func (*EmptyReader) Read(p []byte) (n int, err error) {
	return 0, io.EOF
}

func emptyBody() io.ReadCloser {
	return ioutil.NopCloser(&EmptyReader{})
}
