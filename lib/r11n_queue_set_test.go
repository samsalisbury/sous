package sous

import (
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestR11nQueueSet_PushIfEmpty(t *testing.T) {

	testCases := []struct {
		// PushR11nsWithRepos causes a new R11n to be created with the repo
		// named. Repo is an analogue of "DeploymentID" as different repo always
		// means different DeploymentID, and thus different queue.
		PushR11nsWithRepos []string
		// WantNumQueues is the number of queues we should end up with.
		WantNumQueues int
		WantFinalOK   bool
		// WantFinalQueuedRepo is only checked if WantFinalOK == true.
		WantFinalQueuedRepo string
	}{
		{[]string{"one"}, 1, true, "one"},        // 1 push succeeds
		{[]string{"one", "one"}, 1, false, ""},   // same deployID twice, 2nd fails
		{[]string{"one", "two"}, 2, true, "two"}, // 2 different deployIDs, succeed
	}

	for _, tc := range testCases {
		tc := tc // For the benefit of parallel sub-tests.
		t.Run(strings.Join(tc.PushR11nsWithRepos, "_"), func(t *testing.T) {

			t.Parallel()

			rqs := NewR11nQueueSet()

			// Push the rectifications, ignore all but the last result.
			var gotOK bool
			var gotQR *QueuedR11n
			var lastRepo string
			for _, repoName := range tc.PushR11nsWithRepos {
				r := makeTestR11nWithRepo(repoName)
				gotQR, gotOK = rqs.PushIfEmpty(r)
				lastRepo = repoName
			}

			gotNumQueues := len(rqs.set)
			if gotNumQueues != tc.WantNumQueues {
				t.Errorf("got %d queues; want %d", gotNumQueues, tc.WantNumQueues)
			}

			if gotOK != tc.WantFinalOK {
				t.Errorf("got ok == %t; want %t", gotOK, tc.WantFinalOK)
			}
			if !gotOK {
				if gotQR != nil {
					t.Fatalf("got !ok && qr != nil")
				}
				return // Don't try to check repo on nil qr.
			}
			if err := checkR11nHasRepo(lastRepo)(gotQR); err != nil {
				t.Error(err)
			}

		})
	}
}

// TestR11nQueueSet_Push_async tries to trigger the race detector by calling
// Push repeatedly from multiple goroutines.
// Make sure to run this test with the -race flag!
func TestR11nQueueSet_Push_async(t *testing.T) {

	signal := make(chan struct{})
	go func() {

		const itemCount = 20

		rq := NewR11nQueueSet()

		var wg sync.WaitGroup
		wg.Add(itemCount)
		for i := 0; i < itemCount; i++ {
			i := i
			go func() {
				rq.Push(makeTestR11nWithRepo(strconv.Itoa(i)))
				wg.Done()
			}()
		}
		wg.Wait()

		close(signal)
	}()
	select {
	case <-signal:
	case <-time.After(time.Second):
		t.Errorf("push deadlocked")
	}
}

func TestR11nQueueSet_Wait(t *testing.T) {

	// proceed allows us to control when the queued r11n is processed.
	proceed := make(chan struct{})

	// Set up a queue that processes only on sends to proceed.
	rqs := NewR11nQueueSet(R11nQueueStartWithHandler(func(*QueuedR11n) DiffResolution {
		<-proceed
		return DiffResolution{}
	}))

	// Push a r11n onto the queue.
	qr, ok := rqs.PushIfEmpty(makeTestR11nWithRepo("hi"))
	if !ok {
		t.Fatalf("r11n not queued, cannot proceed with test")
	}

	// Completed marks the queue having processed the r11n pushed above.
	completed := make(chan struct{})
	var completedOK bool
	go func() {
		_, completedOK = rqs.Wait(qr.Rectification.Pair.ID(), qr.ID)
		close(completed)
	}()

	// If completed, fail the test because we didn't send to proceed yet.
	select {
	default:
		// OK
	case <-completed:
		t.Fatalf("r11n processing completed before it should have")
	}

	// Allow the queue handler to proceed.
	proceed <- struct{}{}

	const timeout = 10 * time.Millisecond
	select {
	case <-time.After(timeout):
		t.Fatalf("Wait did not return in under %s after completion", timeout)
	case <-completed:
		// OK, just check we completed OK.
		if !completedOK {
			t.Fatalf("Wait failed for DeployID: %q; R11nID: %q", qr.Rectification.Pair.ID(), qr.ID)
		}
	}
}
