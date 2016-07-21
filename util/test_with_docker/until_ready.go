package test_with_docker

import (
	"fmt"
	"log"
	"time"
)

type (
	// A ReadyFn is a high-order function that returns a predicate to test for
	// readiness and a deferred cleanup function
	ReadyFn func() (desc string, test func() bool, defr func())
	blank   struct{}

	// A ReadyError is returned when UntilReady doesn't get a universal ready result
	ReadyError struct {
		message string
		Errs    []error
	}
)

func (re *ReadyError) Error() string {
	str := re.message
	for _, e := range re.Errs {
		str = str + "\n" + e.Error()
	}
	return str
}

// UntilReady waits for a series of conditions to be true, with optional cleanup
func UntilReady(d, max time.Duration, fns ...ReadyFn) error {
	readies := make(chan error, len(fns))
	done := make(chan blank)
	re := new(ReadyError)

	for _, fn := range fns {
		go loopUntilReady(done, readies, d, fn)
	}

	waitCount := len(fns)
	waitCount = fanInReady(waitCount, readies, max, re, fmt.Sprintf("Still unready after %s ", max))
	close(done)

	fanInReady(waitCount, readies, 3*d, re, "Still waiting on shutdown")
	close(readies)

	if len(re.Errs) > 0 {
		return re
	}

	return nil
}

func sendPanicAsError(out chan error) {
	rec := recover()
	if rec == nil {
		out <- nil
		return
	}
	if err, ok := rec.(error); ok {
		out <- err
		return
	}
	out <- fmt.Errorf("%v", rec)
}

func loopUntilReady(done chan blank, out chan error, pause time.Duration, rfn ReadyFn) {
	desc, test, defr := rfn()
	defer func() { recover() }()
	defer sendPanicAsError(out)
	defer defr()

	for {
		select {
		case <-done:
			out <- fmt.Errorf("Not ready: %s", desc)
			return
		default:
			if test() {
				return
			}
		}

		time.Sleep(pause)
	}
}

func fanInReady(waitCount int, readies chan error, to time.Duration, re *ReadyError, exMsg string) int {
	timeout := time.After(to)
	for waitCount > 0 {
		select {
		case err := <-readies:
			waitCount--
			log.Printf("%v %T", err, err)
			if err != nil {
				log.Print(re.Errs)
				re.Errs = append(re.Errs, err)
			}
		case <-timeout:
			re.message = re.message + exMsg
			break
		}
	}
	return waitCount
}
