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
	Opts
	// Dest is the destination writer.
	Dest io.Writer
	// Writer is the receiving Writer which is prefixed and written to Dest.
	io.Writer
}

// Opts configures your PrefixPipe.
type Opts struct {
	// LogFunc, if set, receives error logs. Defaults to log.Printf from stdlib.
	LogFunc func(string, ...interface{})
}

// DefaultOpts returns the default Opts.
func DefaultOpts() Opts {
	return Opts{
		LogFunc: log.Printf,
	}
}

// New returns a new PrefixPipe that prefixes every line written with the
// formatted string provided and writes it to dest.
//
// Optionally pass one or more config funcs to tweak the options for this
// PrefixPipe, options start as DefaultOpts() and each config func is run
// in the order specified from left to right.
func New(dest io.Writer, prefix string, config ...func(*Opts)) (*PrefixPipe, error) {
	opts := DefaultOpts()
	for _, cfg := range config {
		cfg(&opts)
	}
	r, w, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	pp := &PrefixPipe{
		Opts:   opts,
		Dest:   dest,
		Writer: w,
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
			fmt.Fprintf(pp.Dest, "%s%s\n", prefix, scanner.Text())
		}
	}()
	return pp, nil
}
