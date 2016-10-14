package graph

import (
	"io/ioutil"
	"testing"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
)

func TestBuildGraph(t *testing.T) {
	g := BuildGraph(ioutil.Discard, ioutil.Discard)
	g.Add(&config.Verbosity{})
	g.Add(&config.DeployFilterFlags{})
	g.Add(&config.PolicyFlags{}) //provided by SousBuild
	g.Add(&config.OTPLFlags{})   //provided by SousInit and SousDeploy

	if err := g.Test(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

}

func TestNewBuildConfig(t *testing.T) {
	f := &config.DeployFilterFlags{}
	p := &config.PolicyFlags{}
	bc := &sous.BuildContext{
		Sh: &shell.Sh{},
		Source: sous.SourceContext{
			RemoteURL: "github.com/opentable/present",
			RemoteURLs: []string{
				"github.com/opentable/present",
				"github.com/opentable/also",
			},
			Revision:           "abcdef",
			NearestTagName:     "1.2.3",
			NearestTagRevision: "abcdef",
			Tags: []sous.Tag{
				sous.Tag{Name: "1.2.3"},
			},
		},
	}

	cfg := newBuildConfig(f, p, bc)
	if cfg.Tag != `1.2.3` {
		t.Errorf("Build config's tag wasn't 1.2.3: %#v", cfg.Tag)
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("Not valid build config: %+v", err)
	}

}
