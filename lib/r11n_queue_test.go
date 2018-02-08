package sous

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Test synchronous behaviour of the queue.
func TestR11nQueue_Push_Next_Snapshot_sync(t *testing.T) {
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
				checkR11nHasRepo("one"),
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
				checkR11nHasRepo("one"),
				checkR11nHasRepo("two"),
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
				checkR11nHasRepo("one"),
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
				checkR11nHasRepo("one"),
				checkR11nHasRepo("two"),
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

			// Check snapshot agrees...
			snapshot := rq.Snapshot()
			if gotLen := len(snapshot); gotLen != wantLen {
				t.Errorf("got snapshot len %d; want %d", gotLen, wantLen)
			}
			// Check snapshot order matches queue position.
			lastPos := -1
			for _, qr := range snapshot {
				if qr.Pos <= lastPos {
					t.Errorf("snapshot order does not match queue position")
				}
				lastPos = qr.Pos
			}

			// Iterate over each popped item.
			initialQueueLen := rq.Len()
			for i := 0; i < initialQueueLen; i++ {
				popped := rq.next()
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
					if popped == nil {
						t.Fatalf("got nil QueuedR11n; want not nil")
					}
					if err := tc.WantPoppedOK[i](popped); err != nil {
						t.Error(err)
					}
				})
			}

			// Now the queue is empty, next should block.
			select {
			case <-time.After(10 * time.Millisecond):
				// OK, next blocked for about 10 milliseconds.
			case <-(func() <-chan struct{} {
				c := make(chan struct{})
				go func() {
					defer close(c)
					rq.next()
				}()
				return c
			}()):
				t.Fatal("Next did not block on empty queue.")
			}
		})
	}
}

func TestR11nQueue_ByID(t *testing.T) {
	rq := NewR11nQueue()
	r11nA := makeTestR11nWithRepo("a")
	r11nB := makeTestR11nWithRepo("b")
	qrIDA, ok := rq.Push(r11nA)
	if !ok {
		t.Fatal("setup failed to push r11n")
	}
	qrIDB, ok := rq.Push(r11nB)
	if !ok {
		t.Fatal("setup failed to push r11n")
	}

	gotA, okA := rq.ByID(qrIDA.ID)
	if !okA {
		t.Errorf("got !ok; want ok for item in queue")
	}
	gotRepoA := gotA.Rectification.Pair.ID().ManifestID.Source.Repo
	wantRepoA := "a"
	if gotRepoA != wantRepoA {
		t.Errorf("got r11n with repo %q; want %q", gotRepoA, wantRepoA)
	}

	gotB, okB := rq.ByID(qrIDB.ID)
	if !okB {
		t.Errorf("got !ok; want ok for item in queue")
	}
	gotRepoB := gotB.Rectification.Pair.ID().ManifestID.Source.Repo
	wantRepoB := "b"
	if gotRepoB != wantRepoB {
		t.Errorf("got r11n with repo %q; want %q", gotRepoB, wantRepoB)
	}

	gotC, okC := rq.ByID("nonexistent-id")
	if okC {
		t.Errorf("got ok; want !ok for item not in queue")
	}
	if gotC != nil {
		t.Errorf("got a queued r11n; want nil")
	}
}

func TestR11nQueue_Push_async(t *testing.T) {

	signal := make(chan struct{})
	go func() {
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
		close(signal)
	}()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Errorf("push deadlocked")
	}
}

func TestR11Queue_Next(t *testing.T) {

	rq := NewR11nQueue()
	nextChan := make(chan *QueuedR11n, 1)

	go func() {
		nextChan <- rq.next()
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
	if err := checkR11nHasRepo("hai")(read); err != nil {
		t.Error(err)
	}
}

func TestR11nQueue_PushIfEmpty_sync(t *testing.T) {
	rq := NewR11nQueue()
	pushed, ok := rq.PushIfEmpty(&Rectification{})
	if !ok {
		t.Fatal("PushIfEmpty failed on new queue")
	}
	if pushed == nil {
		t.Fatal("PushIfEmpty returned nil, true")
	}

	notPushed, ok := rq.PushIfEmpty(&Rectification{})
	if ok {
		t.Fatal("PushIfEmpty succeeded on non-empty queue")
	}
	if notPushed != nil {
		t.Fatal("PushIfEmpty failed but returned a non-nil item")
	}
}

func TestR11Queue_PushIfEmpty_async(t *testing.T) {

	rq := NewR11nQueue()

	const itemCount = 100
	var oks, notoks int64
	var wg sync.WaitGroup
	wg.Add(itemCount)
	for i := 0; i < itemCount; i++ {
		go func() {
			defer wg.Done()
			qr, ok := rq.PushIfEmpty(&Rectification{})
			if ok {
				atomic.AddInt64(&oks, 1)
				if qr == nil {
					t.Error("got nil QueuedR11n; want not nil")
				}
				return
			}
			atomic.AddInt64(&notoks, 1)
			if qr != nil {
				t.Errorf("got non-nil QueuedR11n; want nil")
			}
		}()
	}
	wg.Wait()

	if oks != 1 {
		t.Errorf("got %d oks; want 1", oks)
	}
	if notoks != 99 {
		t.Errorf("got %d not oks; want 99", notoks)
	}
}

// makeTestR11nWithRepo creates a test rectification with
// Pair.Post.Deployment.SourceID.Location.Repo == repo.
// This is enough to check identity of the r11n using
// checkPoppedR11nHasRepo.
func makeTestR11nWithRepo(repo string) *Rectification {
	r := &Rectification{
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
	// TODO: Should we not derive name from Pair.Post/Prior instead?
	r.Pair.name = r.Pair.Post.ID()
	return r
}

// checkR11nHasRepo checks identity of r11ns created with
// makeTestR11nWithRepo.
func checkR11nHasRepo(repo string) func(*QueuedR11n) error {
	return func(qr *QueuedR11n) error {
		got := qr.Rectification.Pair.Post.Deployment.SourceID.Location.Repo
		if got != repo {
			return fmt.Errorf("got %q; want %q", got, repo)
		}
		return nil
	}
}
