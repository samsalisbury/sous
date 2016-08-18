/*
Package firsterr provides helpers for getting the first error from a slice of
functions.

The simplest function is Returned:

	func doLotsOfThings() error {
		return firsterr.Returned(doSomething, nextThing, nowDo)
	}

The above assumes that each of the 3 functions doSomething, nextThing, and nowDo
are all of the form 'func() error', which is quite a common signature.
Sometimes, however, these functions need to communicate with each other or set
variables for later use. For that use case we could use Returned, by wrapping
our calls like this:

	func getLotsOfValues() (var1, var2, var3 interface{}, err error) {
		return var1, var2, var3, firsterr.Returned(
			func() err { var1, err = generateValue(); return err },
			func() err { var2, err = useValue(var1); return err },
			func() err { var3, err = useVar2(var2); return err },
		)
	}

However, firsterr also provides the Set function, which passes in a pointer to a
nil error, and checks if that has been set to non-nil, rather than using the
return value. This makes the code shorter and more legible:

	func getLotsOfValues2() (var1, var2, var3 interface{}, err error) {
		return var1, var2, var3, firsterr.Set(
			func(*e error) { var1, e = generateValue() },
			func(*e error) { var2, e = useValue(var1) },
			func(*e error) { var3, e = useVar2(var2) },
		)
	}

When you have lots of functions that don't need to communicate with each other,
you can often safely run them in parallel. This package addresses that situation
as well, with the Parallel() helper, which presents the same interface as
Sequential(), runs all functions in parallel, returning immediately if any of
them return, or set, an error:

	func getLotsOfValues3() (var1, var2, var3 interface{}, err error) {
		return var1, var2, var3, firsterr.Parallel().Set(
			func(e *error) { var1, e := generateValue() },
			func(e *error) { var2, e := generateValue2() },
			func(e *error) { var3, e := generateValue3() },
		)
	}

## Background

In large Go programs, it is common to need to run many functions one after
another, and to require that all of them succeed, or else return the first
error encountered. This can often lead to long, repetitive code, similar to:

	if err := doSomething(); err != nil {
		return err
	}
	if err := nextThing(); err != nil {
		return err
	}
	if err := nowDo(); err != nil {
		return err
	}
	// etc...
	return nil

Whilst highly readable, the repetition here isn't very edifying. The situation
is worse when these functions need to share their results, e.g.:

	var1, err := generateValue()
	if err != nil {
		return nil, err
	}
	var2, err := useValue(var1)
	if err != nil {
		return nil, err
	}
	var3, err := useVar2(var2)
	if err != nil {
		return nil, err
	}
	// etc...
	return nil

This package allows us to write this kind of code with less repetitive noise.
*/
package firsterr

import "sync"

// Set is shorthand for Sequential().Set
func Set(fs ...func(*error)) error {
	return s.Set(fs...)
}

// Returned is shorthand for Sequential().Returned
func Returned(fs ...func() error) error {
	return s.Returned(fs...)
}

// Parallel returns a P, which runs funcs in parallel.
func Parallel() P { return p }

// Sequential returns an S, which runs funcs sequentially.
func Sequential() S { return s }

type (
	// S holds methods for running functions sequentially. When running
	// sequentially, each function is called one at a time. If any of the
	// functions fail, none of the following ones will be run.
	S struct{}
	// P holds methods for running functions in parallel.
	// Note that under P, all functions will be called regardless of any of them
	// failing. There is no cancellation mechanism, therefore, you need to make
	// sure they finish, or you will leak goroutines.
	P struct{}
)

var (
	s S
	p P
)

// Set takes a list of func(*error) and calls them one at a time, in the order
// they were passed in. Each function can optionally set the error pointer
// passed in to an error value.  If the error pointer is non-nil after a
// function completes, Set returns that error and does not run any further
// functions.
func (S) Set(fs ...func(*error)) error {
	for _, f := range fs {
		var err error
		if f(&err); err != nil {
			return err
		}
	}
	return nil
}

// Returned takes a list of func() error and calls them one at a time in the
// order they are passed in. If any of them return a non-nil error, Return
// returns that error, and does not run any further functions.
func (S) Returned(fs ...func() error) error {
	for _, f := range fs {
		if err := f(); err != nil {
			return err
		}
	}
	return nil
}

// Set takes a list of func(*error) and calls them all concurrently. Each
// function can optionally set the error pointer passed in to an error
// value.  If the error pointer is non-nil after a function completes, Set
// immediately returns that error, and abandons the other functions which
// are running in their own goroutines.
func (P) Set(fs ...func(*error)) error {
	wg, errs := pinit(len(fs))
	for _, f := range fs {
		f := f
		go func() {
			defer wg.Done()
			var err error
			if f(&err); err != nil {
				errs <- err
			}
		}()
	}
	return <-errs
}

// Returned takes a list of func() error and calls them all concurrently.
// The moment that any of them return a non-nil error, Return immediately
// returns that error, and abandons the other functions which are running in
// their own goroutines.
func (P) Returned(fs ...func() error) error {
	wg, errs := pinit(len(fs))
	for _, f := range fs {
		f := f
		go func() {
			if err := f(); err != nil {
				errs <- err
			}
			wg.Done()
		}()
	}
	return <-errs
}

func pinit(n int) (*sync.WaitGroup, chan error) {
	wg := &sync.WaitGroup{}
	wg.Add(n)
	errs := make(chan error)
	go func() { wg.Wait(); close(errs) }()
	return wg, errs
}
