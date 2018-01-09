package sous

import (
	"strings"
	"testing"
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
			if err := checkPoppedR11nHasRepo(lastRepo)(gotQR); err != nil {
				t.Error(err)
			}

		})
	}
}
