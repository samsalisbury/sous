package core

import (
	"fmt"
	"net/http"
	"path/filepath"
	"sync"

	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/singularity"
)

type Diff struct {
	Desc       string
	Resolution func(s *singularity.Client) *http.Request
	Error      error
}

func ErrorDiff(err error) Diff {
	return Diff{Error: err}
}

func RequestMissingDiff(requestName string) Diff {
	return Diff{
		Desc: fmt.Sprintf("request %q does not exist", requestName),
	}
}

func (s *MergedState) Diff(dcName string) []Diff {
	dc := s.CompiledDatacentre(dcName)
	c := singularity.NewClient(dc.SingularityURL)
	rs, err := c.Requests()
	if err != nil {
		cli.Fatalf("%s", err)
	}
	cli.Logf("%s: %d", dc.SingularityURL, len(rs))
	return dc.DiffRequests()
}

func (d CompiledDatacentre) DiffRequests() []Diff {
	ds := make(chan []Diff)
	wg := sync.WaitGroup{}
	wg.Add(len(d.Manifests))
	for _, m := range d.Manifests {
		m := m
		go func() {
			ds <- m.Diff(d.SingularityURL)
			wg.Done()
		}()
	}
	go func() { wg.Wait(); close(ds) }()
	result := []Diff{}
	for diffs := range ds {
		result = append(result, diffs...)
	}
	return result
}

func (d DatacentreManifest) Diff(singularityURL string) []Diff {
	s := singularity.NewClient(singularityURL)
	requestName := filepath.Base(d.App.SourceRepo)
	r, err := s.Request(requestName)
	if err != nil {
		return []Diff{ErrorDiff(err)}
	}
	if r == nil {
		return []Diff{RequestMissingDiff(requestName)}
	}
	return nil
}
