package cli

import (
	"flag"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestManifestGetArgs(t *testing.T) {
	fs := flag.NewFlagSet("test-for-manifest-get", flag.ContinueOnError)
	smg := &SousManifestGet{}
	smg.AddFlags(fs)
	fs.Parse([]string{"-repo", "github.com/example/test", "-flavor", "winning"})

	assert.Equal(t, "github.com/example/test", smg.DeployFilterFlags.Repo)
	assert.Equal(t, "winning", smg.DeployFilterFlags.Flavor)
}

func TestManifestSetArgs(t *testing.T) {
	fs := flag.NewFlagSet("test-for-manifest-set", flag.ContinueOnError)
	smg := &SousManifestSet{}
	smg.AddFlags(fs)
	fs.Parse([]string{"-repo", "github.com/example/test", "-flavor", "winning"})

	assert.Equal(t, "github.com/example/test", smg.DeployFilterFlags.Repo)
	assert.Equal(t, "winning", smg.DeployFilterFlags.Flavor)
}

func TestManifestYAML(t *testing.T) {
	uripath := "certainly/i/am/healthy"
	yml, err := yaml.Marshal(sous.ManifestFixture("simple"))
	require.NoError(t, err)
	assert.Regexp(t, "(?i).*checkready.*", string(yml))

	newM := sous.Manifest{}
	err = yaml.Unmarshal(yml, &newM)
	require.NoError(t, err)

	assert.Equal(t, newM.Deployments["ci"].Startup.CheckReadyURIPath, uripath)
}
