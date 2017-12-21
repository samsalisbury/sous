package sous

import (
	"fmt"
	"testing"
)

// Test synchronous behaviour of the queue.
func TestR11nQueue_Push_Pop_sync(t *testing.T) {

	// makeTestR11nWithRepo creates a test rectification with
	// Pair.Post.Deployment.SourceID.Location.Repo == repo.
	// This is enough to check identity of the r11n using
	// checkPoppedR11nHasRepo.
	makeTestR11nWithRepo := func(repo string) *Rectification {
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
	checkPoppedR11nHasRepo := func(repo string) func(*QueuedR11n) error {
		return func(qr *QueuedR11n) error {
			got := qr.Rectification.Pair.Post.Deployment.SourceID.Location.Repo
			if got != repo {
				return fmt.Errorf("got %q; want %q", got, repo)
			}
			return nil
		}
	}

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
