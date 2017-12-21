package sous

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test synchronous behaviour of the queue.
func TestR11nQueue_Push_Pop_sync(t *testing.T) {
	testCases := []struct {
		// Desc is a short description of the test.
		Desc string
		// Init is the queue to start with for this test.
		Init *R11nQueue
		// WantLen is the length we expect the queue to be after all items in
		// Push have been pushed onto it. If zero, this defaults to the number
		// of items in Push.
		WantLen int
		// Push is a slice of R11ns that are pushed onto the queue one at a
		// time.
		Push []*Rectification
		// WantPushed funcs are run one at a time after each Push call, and
		// passed the returned values from Push. If they return an error, the
		// test has the error added.
		WantPushed []func(*QueuedR11n, bool) error
		// WantPoppedOK funcs are run one at a time after calling Pop.
		// If Pop returns nil or false, the corresponding WantPoppedOK is not
		// run, and a test error is generated instead.
		WantPoppedOK []func(*QueuedR11n) error
	}{
		{
			Desc: "default empty queue; no Push",
			Init: NewR11nQueue(),
		},
		{
			Desc: "default queue; Push one item",
			Init: NewR11nQueue(),
			Push: []*Rectification{
				makeTestR11nWithRepo("one"),
			},
			WantPoppedOK: []func(*QueuedR11n) error{
				checkPoppedR11nHasRepo("one"),
			},
		},
		{
			Desc: "default queue; Push two items",
			Init: NewR11nQueue(),
			Push: []*Rectification{
				makeTestR11nWithRepo("one"),
				makeTestR11nWithRepo("two"),
			},
			WantPoppedOK: []func(*QueuedR11n) error{
				checkPoppedR11nHasRepo("one"),
				checkPoppedR11nHasRepo("two"),
			},
		},
		{
			Desc: "queue cap one; Push two items",
			Init: NewR11nQueue(R11nQueueCap(1)),
			Push: []*Rectification{
				makeTestR11nWithRepo("one"),
				makeTestR11nWithRepo("two"),
			},
			WantLen: 1, // Can't go above capacity.
			WantPushed: []func(*QueuedR11n, bool) error{
				nil, // No custom check for the first one.
				func(qr *QueuedR11n, ok bool) error {
					if qr != nil {
						return fmt.Errorf("got QueuedR11n %s; want nil", qr.ID)
					}
					if ok {
						return fmt.Errorf("got ok true; want false")
					}
					return nil
				},
			},
			WantPoppedOK: []func(*QueuedR11n) error{
				checkPoppedR11nHasRepo("one"),
			},
		},
		{
			Desc: "queue cap two; Push three items",
			Init: NewR11nQueue(R11nQueueCap(2)),
			Push: []*Rectification{
				makeTestR11nWithRepo("one"),
				makeTestR11nWithRepo("two"),
				makeTestR11nWithRepo("three"),
			},
			WantLen: 2, // Can't go above capacity.
			WantPushed: []func(*QueuedR11n, bool) error{
				nil,
				nil, // No custom checks for the first or second ones.
				func(qr *QueuedR11n, ok bool) error {
					if qr != nil {
						return fmt.Errorf("got QueuedR11n %s; want nil", qr.ID)
					}
					if ok {
						return fmt.Errorf("got ok true; want false")
					}
					return nil
				},
			},
			WantPoppedOK: []func(*QueuedR11n) error{
				checkPoppedR11nHasRepo("one"),
				checkPoppedR11nHasRepo("two"),
			},
		},
	}

	for _, tc := range testCases {
		tc := tc // Needed since we're using t.Parallel below.
		t.Run(tc.Desc, func(t *testing.T) {
			// Top-level can be run in parallel, further sub-tests must not be
			// as they rely on pushing and popping in a certain order.
			t.Parallel()
			rq := tc.Init
			pushedIDsOrder := make(map[R11nID]int, 10)
			// Push each item to be pushed.
			for i, r11n := range tc.Push {
				desc := fmt.Sprintf("pushed %d", i)
				t.Run(desc, func(t *testing.T) {

					pushed, ok := rq.Push(r11n)

					// If there is a non-nil check for this Push call, run it.
					if len(tc.WantPushed) > i && tc.WantPushed[i] != nil {
						if err := tc.WantPushed[i](pushed, ok); err != nil {
							t.Error(err)
						}
					}

					// If push wasn't ok, stop running checks.
					if !ok || pushed == nil {
						return
					}

					// Check each ID is unique.
					if _, ok := pushedIDsOrder[pushed.ID]; ok {
						t.Errorf("non-unique ID: %q", pushed.ID)
					}
					pushedIDsOrder[pushed.ID] = i

					// Check that positions increment each time.
					if pushed.Pos != i {
						t.Errorf("got pos %d; want %d", pushed.Pos, i)
					}

				})
			}

			// Check length after pushing all items.
			gotLen, wantLen := rq.Len(), tc.WantLen
			if wantLen == 0 {
				// Default to length of Push if tc.WantLen is zero.
				wantLen = len(tc.Push)
			}
			if gotLen != wantLen {
				t.Errorf("got len %d; want %d", gotLen, wantLen)
			}

			// Iterate over each popped item.
			i := 0
			for popped, ok := rq.Pop(); ok; popped, ok = rq.Pop() {
				desc := fmt.Sprintf("popped %d", i)
				t.Run(desc, func(t *testing.T) {

					// Check popped always has position -1.
					if popped.Pos != -1 {
						t.Errorf("got position %d; want %d", popped.Pos, -1)
					}

					// Check popped order is the same as pushed order.
					if pushedOrder, ok := pushedIDsOrder[popped.ID]; ok {
						if pushedOrder != i {
							t.Fatalf("popped %q at %d; want it at %d",
								popped.ID, i, pushedOrder)
						}
					} else {
						t.Fatalf("popped un-pushed ID %q", popped.ID)
					}

					// If there is a check for this popped item, run it.
					if len(tc.WantPoppedOK) <= i {
						return
					}
					// Popped assertions always assume we get something popped.
					if !ok {
						t.Fatalf("got !ok; want ok for WantPoppedOK")
					}
					if popped == nil {
						t.Fatalf("got nil QueuedR11n; want not nil")
					}
					if err := tc.WantPoppedOK[i](popped); err != nil {
						t.Error(err)
					}
				})

				i++
			}

			// Now the queue is empty, Pop should return nil, false.
			popped, ok := rq.Pop()
			if popped != nil {
				t.Errorf("got QueuedR11n %s; want nil", popped.ID)
			}
			if ok {
				t.Errorf("got ok true; want false")
			}
		})
	}
}

func TestR11nQueue_Push_async(t *testing.T) {

	// Make sure to run this test with the -race flag!

	const queueSize = 10
	const itemCount = 20

	rq := NewR11nQueue(R11nQueueCap(queueSize))

	// oks collects the number of oks received from Push.
	var oks int64

	var wg sync.WaitGroup
	wg.Add(itemCount)
	for i := 0; i < itemCount; i++ {
		go func() {
			_, ok := rq.Push(&Rectification{})
			if ok {
				atomic.AddInt64(&oks, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if oks != queueSize {
		t.Errorf("got %d oks; want %d", oks, queueSize)
	}

}

func TestR11nQueue_Pop_async(t *testing.T) {

	// Make sure to run this test with the -race flag!

	const queueSize = 10
	const itemCount = 20

	rq := NewR11nQueue(R11nQueueCap(queueSize))
	for i := 0; i < queueSize; i++ {
		rq.Push(&Rectification{})
	}

	// oks collects the number of oks received from Push.
	var oks int64

	var wg sync.WaitGroup
	wg.Add(itemCount)
	for i := 0; i < itemCount; i++ {
		go func() {
			_, ok := rq.Pop()
			if ok {
				atomic.AddInt64(&oks, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if oks != queueSize {
		t.Errorf("got %d oks; want %d", oks, queueSize)
	}

}

func TestR11Queue_Next(t *testing.T) {

	rq := NewR11nQueue()
	nextChan := make(chan *QueuedR11n, 1)

	go func() {
		nextChan <- rq.Next()
	}()

	// Nasty sleep, want to prove rq.Next hasn't put anything on the queue.
	time.Sleep(100 * time.Millisecond)

	if len(nextChan) != 0 {
		t.Fatalf("Next returned before Push was called")
	}

	rq.Push(makeTestR11nWithRepo("hai"))

	// Eeegh.
	time.Sleep(100 * time.Millisecond)
	if len(nextChan) != 1 {
		t.Fatalf("Next did not read written within 100ms")
	}

	read := <-nextChan
	if err := checkPoppedR11nHasRepo("hai")(read); err != nil {
		t.Error(err)
	}
}

func TestR11Queue_Next_Pop_race(t *testing.T) {

	// This test ensures that if using both Next and Pop, there are
	// no data races. Ideally Next and Pop should receive about 50%
	// of the items each.

	rq := NewR11nQueue()

	const itemCount = 1000

	var pops, nexts int64

	var wg sync.WaitGroup
	wg.Add(2 * itemCount) // 2* because we want all pushes, and reads to finish.

	go func() {
		// Try to pop in a hot loop.
		for _, ok := rq.Pop(); ; _, ok = rq.Pop() {
			if ok {
				atomic.AddInt64(&pops, 1)
				wg.Done()
			}
		}
	}()
	go func() {
		// Read next as fast as possible.
		for rq.Next(); ; rq.Next() {
			atomic.AddInt64(&nexts, 1)
			wg.Done()
		}
	}()

	go func() {
		for i := 0; i < itemCount; i++ {
			if _, ok := rq.Push(&Rectification{}); !ok {
				i--
				continue
			}
			wg.Done()
		}
	}()

	wg.Wait()

	// Strictly speaking this could happen even with this working,
	// but sample size of 1000 means this is likely to be split 50/50.
	if pops == 0 {
		t.Errorf("Pop received no items")
	}
	if nexts == 0 {
		t.Errorf("Next received no items")
	}

	if pops+nexts != itemCount {
		t.Errorf("got %d pops and %d nexts (want: %d)", pops, nexts, pops+nexts)
	} else {
		t.Logf("success: got %d pops and %d nexts (total: %d)", pops, nexts, pops+nexts)
	}

}

// makeTestR11nWithRepo creates a test rectification with
// Pair.Post.Deployment.SourceID.Location.Repo == repo.
// This is enough to check identity of the r11n using
// checkPoppedR11nHasRepo.
func makeTestR11nWithRepo(repo string) *Rectification {
	return &Rectification{
		Pair: DeployablePair{
			Post: &Deployable{
				Status: 0,
				Deployment: &Deployment{
					SourceID: SourceID{
						Location: SourceLocation{
							Repo: repo,
						},
					},
				},
			},
		},
	}
}

// checkPoppedR11nHasRepo checks identity of r11ns created with
// makeTestR11nWithRepo.
func checkPoppedR11nHasRepo(repo string) func(*QueuedR11n) error {
	return func(qr *QueuedR11n) error {
		got := qr.Rectification.Pair.Post.Deployment.SourceID.Location.Repo
		if got != repo {
			return fmt.Errorf("got %q; want %q", got, repo)
		}
		return nil
	}
}
