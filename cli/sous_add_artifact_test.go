package cli

import (
	"testing"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
	"github.com/stretchr/testify/require"
)

func TestAddArtifact_Help(t *testing.T) {

	p := &SousArtifactAdd{}

	help := p.Help()

	require.True(t, len(help) > 0)

}

func TestAddArtifact_Execute(t *testing.T) {

	gr := graph.DefaultTestGraph(t)

	require := require.New(t)
	a := &SousArtifactAdd{SousGraph: gr}

	args := []string{"", ""}

	result := a.Execute(args)

	require.IsType(cmdr.UsageErr{}, result)

}
