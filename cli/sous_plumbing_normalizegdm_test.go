package cli

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHelp(t *testing.T) {

	p := &SousPlumbingNormalizeGDM{}

	help := p.Help()

	require.True(t, len(help) > 0)

}
