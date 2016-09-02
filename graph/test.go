package graph

import (
	"io/ioutil"
	"testing"
)

func TestBuildGraph(t *testing.T) {

	g := BuildGraph(&CLI{}, ioutil.Discard, ioutil.Discard)
	g.Add(&Sous{})
	g.Add(&DeployFilterFlags{})
	g.Add(&PolicyFlags{}) //provided by SousBuild
	g.Add(&OTPLFlags{})   //provided by SousInit and SousDeploy

	if err := g.Test(); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

}
