package smoke

import (
	"log"
	"testing"
	"time"

	sing "github.com/opentable/go-singularity"
)

// ResetSingularity clears out the state from the integration singularity service
// Call it (with and extra call deferred) anywhere integration tests use Singularity
func resetSingularity(t *testing.T, singularityURL string) {
	const pollLimit = 30
	const retryLimit = 3
	t.Log("Resetting Singularity...")
	singClient := sing.NewClient(singularityURL)

	reqList, err := singClient.GetRequests(false)
	if err != nil {
		panic(err)
	}

	// Singularity is sometimes not actually deleting a request until the second attempt...
	for j := retryLimit; j >= 0; j-- {
		for _, r := range reqList {
			_, err := singClient.DeleteRequest(r.Request.Id, nil)
			if err != nil {
				panic(err)
			}
		}

		log.Printf("Singularity resetting: Issued deletes for %d requests. Awaiting confirmation they've quit.", len(reqList))

		for i := pollLimit; i > 0; i-- {
			reqList, err = singClient.GetRequests(false)
			if err != nil {
				panic(err)
			}
			if len(reqList) == 0 {
				log.Printf("Singularity successfully reset.")
				return
			}
			time.Sleep(time.Second)
		}
	}
	for n, req := range reqList {
		log.Printf("Singularity reset failure: stubborn request: #%d/%d %#v", n+1, len(reqList), req)
	}
	t.Fatalf("singularity not reset after %d * %d tries - %d requests remain", retryLimit, pollLimit, len(reqList))
}
