package prefixpipe

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
)

// PrefixPipe is an io.Writer with some extra settings.
type PrefixPipe struct {
	// Writer is the receiving Writer which is prefixed and written to Out.
	io.Writer
	// Out is the destination writer, defaults to os.Stdout.
	Out io.Writer
	// LogFunc, if set, receives error logs. Defaults to log.Printf from stdlib.
	LogFunc func(string, ...interface{})
}

// New returns a new PrefixPipe that prefixes every line written with the
// formatted string provided.
func New(prefixFormat string, a ...interface{}) (*PrefixPipe, error) {
	prefix := fmt.Sprintf(prefixFormat, a...)
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	pp := &PrefixPipe{
		Writer:  w,
		Out:     os.Stdout,
		LogFunc: log.Printf,
	}
	go func() {
		defer func() {
			if err := r.Close(); err != nil {
				pp.LogFunc("Failed to close reader: %s", err)
			}
			if err := w.Close(); err != nil {
				pp.LogFunc("Failed to close writer: %s", err)
			}
		}()
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				pp.LogFunc("Error prefixing: %s", err)
			}
			fmt.Fprintf(pp.Out, "%s%s\n", prefix, scanner.Text())
		}
	}()
	return pp, nil
}
