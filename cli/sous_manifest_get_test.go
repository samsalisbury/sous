package cli

import (
	"fmt"
	"testing"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

func TestSousManifestGet_Execute(t *testing.T) {

	badResolveFilter := graph.NewSousGraph()
	badResolveFilter.Add(
		func() (*graph.RefinedResolveFilter, error) {
			return &graph.RefinedResolveFilter{}, fmt.Errorf("an error")
		},
		graph.TargetManifestID{},
		graph.HTTPClient{HTTPClient: &restful.DummyHTTPClient{}},
		graph.LogSink{LogSink: logging.SilentLogSet()},
	)

	badHTTPClient := graph.NewSousGraph()
	badHTTPClient.Add(
		&graph.RefinedResolveFilter{},
		graph.TargetManifestID{},
		graph.HTTPClient{HTTPClient: &restful.DummyHTTPClient{
			AlwaysReturnErr: fmt.Errorf("an error"),
		}},
		graph.LogSink{LogSink: logging.SilentLogSet()},
	)

	okGraph := graph.NewSousGraph()
	okGraph.Add(
		&graph.RefinedResolveFilter{},
		graph.TargetManifestID{},
		graph.HTTPClient{HTTPClient: &restful.DummyHTTPClient{}},
		graph.LogSink{LogSink: logging.SilentLogSet()},
	)

	assertExitCode := func(t *testing.T, g *graph.SousGraph, want int) {
		t.Helper()
		c := &SousManifestGet{SousGraph: g}
		got := c.Execute(nil).ExitCode()
		if got != want {
			t.Errorf("got exit code %d; want %d", got, want)
		}
	}

	const ok, internalError = 0, 255

	assertExitCode(t, okGraph, ok)
	assertExitCode(t, badResolveFilter, internalError)
	assertExitCode(t, badHTTPClient, internalError)
}
