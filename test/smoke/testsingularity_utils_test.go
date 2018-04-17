package smoke

import (
	"log"
	"testing"
	"time"

	sing "github.com/opentable/go-singularity"
)

type Singularity struct {
	URL    string
	client *sing.Client
}

func NewSingularity(baseURL string) *Singularity {
	return &Singularity{URL: baseURL, client: sing.NewClient(baseURL)}
}

func (s *Singularity) Reset(t *testing.T) {
	t.Helper()
	const pollLimit = 30
	const retryLimit = 3
	t.Log("Resetting Singularity...")

	reqList, err := s.client.GetRequests(false)
	if err != nil {
		panic(err)
	}

	// Singularity is sometimes not actually deleting a request until the second attempt...
	for j := retryLimit; j >= 0; j-- {
		for _, r := range reqList {
			_, err := s.client.DeleteRequest(r.Request.Id, nil)
			if err != nil {
				panic(err)
			}
		}

		log.Printf("Singularity resetting: Issued deletes for %d requests. Awaiting confirmation they've quit.", len(reqList))

		for i := pollLimit; i > 0; i-- {
			reqList, err = s.client.GetRequests(false)
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
