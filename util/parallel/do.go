package parallel

import "sync"

// Do takes a list of func(*error) and calls them all concurrently. Each
// function can optionally set the error pointer passed in to an error value.
// If the error pointer is non-nil after a function completes, Do immediately
// returns that error, and abandons the other functions which are running in
// their own goroutines.
func Do(fs ...func(*error)) error {
	wg := sync.WaitGroup{}
	wg.Add(len(fs))
	errs := make(chan error)
	go func() { wg.Wait(); close(errs) }()
	for _, f := range fs {
		f := f
		go func() {
			var err error
			f(&err)
			if err != nil {
				errs <- err
			}
			wg.Done()
		}()
	}
	return <-errs
}
