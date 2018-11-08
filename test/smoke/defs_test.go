//+build smoke

package smoke

import (
	"strings"
	"testing"

	"github.com/opentable/sous/util/testmatrix"
)

func TestDefs(t *testing.T) {

	m := newRunner(t, matrix())

	m.RunScenario("no-allowed-advisories", func(t *testing.T, s testmatrix.Scenario, lf *testmatrix.LateFixture) {
		f := newConfiguredFixture(t, s, func(c *fixtureConfig) {
			for _, name := range c.InitialState.Defs.Clusters.Names() {
				cl := c.InitialState.Defs.Clusters[name]
				cl.AllowedAdvisories = []string{}
				c.InitialState.Defs.Clusters[name] = cl
			}
		})
		lf.Set(f)

		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{kind: "http-service", tag: "1.2.3", cluster: "cluster1", repo: p.repo}

		initBuild(t, p, flags, setMinimalMemAndCPUNumInst1)

		stderr := p.MustFail(t, "deploy", flags.SousDeployFlags())
		want := "Advisory unacceptable on image:"
		if !strings.Contains(stderr, want) {
			t.Errorf("got stderr %q; want it to contain %q", stderr, want)
		}
	})

}
