package prefixpipe

import (
	"bufio"
	"fmt"
	"io"
	"log"
)

// PrefixPipe prefixes all lines written to it, and writes them to Dest.
type PrefixPipe struct {
	Opts
	// Dest is the destination writer.
	Dest io.Writer
	// WriterCloser is the receiving Writer which is prefixed and written to
	// Dest. Remember to call Close() when you're done, or you will leak a
	// goroutine.
	io.WriteCloser
	errs chan error
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
//
// To ensure you don't leak goroutines, you must call Close() on the returned
// *PrefixPipe. To ensure every line gets written to Dest, you must call Wait()
// after Close().
func New(dest io.Writer, prefix string, config ...func(*Opts)) *PrefixPipe {
	opts := DefaultOpts()
	for _, cfg := range config {
		cfg(&opts)
	}
	r, w := io.Pipe()
	pp := &PrefixPipe{
		Opts:        opts,
		Dest:        dest,
		WriteCloser: w,
		errs:        make(chan error),
	}
	go func() {
		defer func() {
			defer close(pp.errs)
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
				break
			}
			fmt.Fprintf(pp.Dest, "%s%s\n", prefix, scanner.Text())
		}
	}()

	return pp
}

// Wait waits for the pipe to be closed and to flush all its contents.
func (pp *PrefixPipe) Wait() error {
	var errs []error
	for err := range pp.errs {
		errs = append(errs, err)
	}
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}
	return fmt.Errorf("multiple errors encountered: % #v", errs)
}
