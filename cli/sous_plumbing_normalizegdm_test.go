package cli

import (
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
	"github.com/stretchr/testify/require"
)

func TestHelp(t *testing.T) {

	p := &SousPlumbingNormalizeGDM{}

	help := p.Help()

	require.True(t, len(help) > 0)

}

func TestExecute(t *testing.T) {

	gr := graph.DefaultTestGraph()

	c := &config.Config{Server: "", StateLocation: "/tmp/sous"}

	require := require.New(t)
	p := &SousPlumbingNormalizeGDM{SousGraph: gr, LocalSousConfig: graph.LocalSousConfig{Config: c}}

	args := []string{"", ""}

	result := p.Execute(args)

	require.IsType(cmdr.UnknownErr{}, result)

}
