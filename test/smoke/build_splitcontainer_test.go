//+build smoke

package smoke

import (
	"testing"
)

func TestSplitContainer(t *testing.T) {

	pf := pfs.newParallelTestFixture(t)

	fixtureConfigs := []fixtureConfig{
		{dbPrimary: false},
		{dbPrimary: true},
	}

	pf.RunMatrix(fixtureConfigs,
		PTest{Name: "simple-splitcontainer", Test: func(t *testing.T, f *TestFixture) {
			client := f.setupProject(t, simpleServerSplitContainer())
			client.MustRun(t, "init", nil, "-kind", "http-service")
			client.MustRun(t, "build", nil, "-tag", "1")
			client.MustRun(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1")
		}},
	)
}
