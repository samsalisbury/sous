package coaxer

import (
	"context"
	"fmt"
	"time"
)

type (
	// Coaxer tries to coax a value into existence.
	Coaxer struct {
		// Attempts is the maximum number of times to try calling the manifest func
		// before giving up.
		Attempts int
		// Backoff is the current backoff duration (time between attempts).
		Backoff time.Duration
		// BackoffScale is the scaling factor used on Backoff on each attempt.
		BackoffScale int
		// Timeout is how long manifest is given per attempt before being abandoned.
		Timeout time.Duration
		// TimeoutScale is the scaling factor for the Timeout on each attempt.
		TimeoutScale int
		// DebugFunc is called synchronously before and after manifest is called.
		DebugFunc func(desc string)
	}

	// Result is the result of coaxing.
	Result struct {
		// Value represents the final value obtained by coaxing. If Error is not
		// nil, then Value will be nil.
		Value interface{}
		// Error represents the fact that Value was not successfully coaxed into
		// existence. If error is not nil, then Value is nil.
		Error error
	}

	// A ManifestFunc is specified by the consumer of this package to describe how
	// to optimistically generate a value, or return an error should that not be
	// possible.
	ManifestFunc func() (interface{}, error)
)

const (
	// DefaultAttempts is the default number of attempts for a new Coaxer.
	DefaultAttempts = 3
	// DefaultBackoffScale is the default BackoffScale for a new Coaxer.
	DefaultBackoffScale = 2
	// DefaultTimeoutScale is the default TimeoutScale for a new Coaxer.
	DefaultTimeoutScale = 1
	// DefaultBackoff is the default initial Backoff duration for a new Coaxer.
	DefaultBackoff = 100 * time.Millisecond
	// DefaultTimeout is the default initial timeout for each attempt.
	DefaultTimeout = 2 * time.Second
)

// NewCoaxer returns a new configured coaxer.
//
// You can optionally pass any number of functions to configure which will
// be called in the order passed on the new Coaxer before it is returned.
func NewCoaxer(configure ...func(*Coaxer)) *Coaxer {
	c := &Coaxer{
		Attempts:     DefaultAttempts,
		Backoff:      DefaultBackoff,
		BackoffScale: DefaultBackoffScale,
		Timeout:      DefaultTimeout,
		TimeoutScale: DefaultTimeoutScale,
		DebugFunc:    func(desc string) {},
	}
	for _, f := range configure {
		if f != nil {
			f(c)
		}
	}
	return c
}

// Coax immediately returns a channel that will eventually produce the Result of
// this attempt to coax the result of manifest into existence.
//
// The descf and a... parameters are passed to fmt.Sprintf to populate the
// description of what the manifest function does. This is used only for logging
// and error messages.
func (c *Coaxer) Coax(ctx context.Context, manifest ManifestFunc, descf string, a ...interface{}) Promise {
	run := coaxRun{
		Coaxer:      *c, // This is a copy, so don't worry about modifying it.
		ctx:         ctx,
		desc:        fmt.Sprintf(descf, a...),
		manifest:    manifest,
		result:      make(chan Result),
		finalResult: make(chan Result),
		debugFunc:   c.DebugFunc,
	}
	return run.future()
}
