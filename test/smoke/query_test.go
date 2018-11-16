//+build smoke

package smoke

import (
	"strings"
	"testing"
)

func assumeSuccessfullyDeployed(t *testing.T, f *fixture, p *sousProject, flags *sousFlags, reqID string) {
	initBuildDeploy(t, p, flags, setMinimalMemAndCPUNumInst1)
	assertActiveStatus(t, f, reqID)
	assertSingularityRequestTypeService(t, f, reqID)
	assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
}

func TestQuery(t *testing.T) {

	m := newRunner(t, matrix())
	m.Run("gdm-filters", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{kind: "http-service", tag: "1.2.3", cluster: "cluster1", repo: p.repo}
		reqID := f.DefaultSingReqID(t, flags)

		assumeSuccessfullyDeployed(t, f, p, flags, reqID)

		cases := []struct {
			filters   string
			wantCount int
		}{
			{"", 1},
			{"hasowners=true", 1},
			{"hasowners=false", 0},
			{"hasimage=true", 1},
			{"hasimage=false", 0},
			{"zeroinstances=true", 0},
			{"zeroinstances=false", 1},
		}

		for _, c := range cases {
			t.Run(c.filters, func(t *testing.T) {
				got := p.MustRun(t, "query gdm", nil, "-format", "json",
					"-filters", c.filters)
				count := len(strings.Split(got, "\n"))
				if count != c.wantCount {
					t.Errorf("filter %q got %d results, want %d",
						c.filters, count, c.wantCount)
				}
			})
		}
	})
}
