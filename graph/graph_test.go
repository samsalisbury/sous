package graph

import (
	"io/ioutil"
	"testing"

	"github.com/opentable/sous/config"
)

func TestBuildGraph(t *testing.T) {
	g := BuildGraph(&CLI{}, ioutil.Discard, ioutil.Discard)
	g.Add(&config.Verbosity{})
	g.Add(&config.DeployFilterFlags{})
	g.Add(&config.PolicyFlags{}) //provided by SousBuild
	g.Add(&config.OTPLFlags{})   //provided by SousInit and SousDeploy

	if err := g.Test(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

}
