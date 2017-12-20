package sous

import (
	"fmt"
	"testing"
)

// Test synchronous behaviour of the queue.
func TestR11nQueue_synchronous(t *testing.T) {

	testCases := []struct {
		// Desc is a short description of the test.
		Desc string
		// WantLen is the expected result of Len after all the items in Push
		// have been pushed.
		WantLen int
		// Push is a slice of R11ns that are pushed onto the queue one at a
		// time.
		Push []*Rectification
		// AssertPushed is called on each item returned from Push calls.
		WantPushed,
		// AssertPeek funcs are run one at a time and passed the result of
		// Peeking at the next item, which is then popped.
		WantPeeked []func(*QueuedR11n) error
	}{
		{
			Desc:    "new queue",
			WantLen: 0,
		},
		{
			Desc: "one zero item",
			Push: []*Rectification{
				&Rectification{},
			},
			WantLen: 1,
		},
		{
			Desc: "two zero items",
			Push: []*Rectification{
				&Rectification{},
				&Rectification{},
			},
			WantLen: 2,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Desc, func(t *testing.T) {
			// Top-level can be run in parallel, further sub-tests must not be
			// as they rely on pushing and popping in a certain order.
			t.Parallel()
			rq := NewR11nQueue()
			pushedIDsOrder := make(map[R11nID]int, 10)
			// Push each item to be pushed.
			for i, r11n := range tc.Push {
				desc := fmt.Sprintf("pushed %d", i)
				t.Run(desc, func(t *testing.T) {
					pushed := rq.Push(r11n)

					// Check each ID is unique.
					if _, ok := pushedIDsOrder[pushed.ID]; ok {
						t.Errorf("non-unique ID: %q", pushed.ID)
					}
					pushedIDsOrder[pushed.ID] = i

					// Check that positions increment each time.
					if pushed.Pos != i {
						t.Errorf("got pos %d; want %d", pushed.Pos, i)
					}

					// If there is an assertion for this pushed item, run it.
					if len(tc.WantPushed) <= i {
						return
					}
					if err := tc.WantPushed[i](pushed); err != nil {
						t.Error(err)
					}
				})
			}

			// Check length after pushing all items.
			gotLen, wantLen := rq.Len(), tc.WantLen
			if gotLen != wantLen {
				t.Errorf("got len %d; want %d", gotLen, wantLen)
			}

			// Iterate over each peeked item.
			i := 0
			for peeked, ok := rq.Pop(); ok; peeked, ok = rq.Pop() {
				desc := fmt.Sprintf("popped %d", i)
				t.Run(desc, func(t *testing.T) {

					// Check peeked always has position 0.
					if peeked.Pos != 0 {
						t.Errorf("got position %d; want %d", peeked.Pos, 0)
					}

					// Check peeked order is the same as pushed order.
					if pushedOrder, ok := pushedIDsOrder[peeked.ID]; ok {
						if pushedOrder != i {
							t.Fatalf("popped %q at %d; want it at %d",
								peeked.ID, i, pushedOrder)
						}
					} else {
						t.Fatalf("popped un-pushed ID %q", peeked.ID)
					}

					// If there is an assertion for this peeked item, run it.
					if len(tc.WantPeeked) <= i {
						return
					}
					if err := tc.WantPeeked[i](peeked); err != nil {
						t.Error(err)
					}
				})

				i++
			}
		})
	}
}
