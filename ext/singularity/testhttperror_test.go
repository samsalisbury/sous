package singularity

import (
	"fmt"
	"log"
	"path/filepath"
	"runtime"
)

type httpError struct {
	Code int
	Text string
}

func (h *httpError) Error() string   { return fmt.Sprintf("HTTP %d: %s", h.Code, h.Text) }
func (h *httpError) Temporary() bool { return true }

func httpErr(code int, format string, a ...interface{}) error {
	err := &httpError{Code: 404, Text: fmt.Sprintf(format, a...)}
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Panicf("httpErr unable to get its caller")
	}
	file = filepath.Base(file)
	log.Printf("%s:%d: %s", file, line, err)
	return err
}
