//+build smoke

package smoke

import (
	"strings"
	"testing"
)

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
			// There are 12 total deployments to start with:
			//   3 manifests x 3 clusters in initial state +
			//   1 manifest x 3 clusters created by the test.
			//
			// TODO SS: Make initial state explicit in tests like this where it
			//          greatly affects the output.
			{"", 12},
			{"hasowners=true", 0},
			{"hasowners=false", 12},
			{"hasimage=true", 1},
			{"hasimage=false", 11},
			{"zeroinstances=true", 0},
			{"zeroinstances=false", 12},
			{"hasowners=false hasimage=true", 1},
			{"zeroinstances=false hasimage=true", 1},
			{"hasowners=false zeroinstances=true", 0},
			{"hasowners=true zeroinstances=true", 0},
		}

		for _, c := range cases {
			t.Run(c.filters, func(t *testing.T) {
				got := p.MustRun(t, "query gdm", nil, "-format", "json",
					"-filters", c.filters)
				got = strings.TrimSpace(got)
				lines := strings.Split(got, "\n")
				var nonemptyLines []string
				for _, l := range lines {
					l = strings.TrimSpace(l)
					if l == "" {
						continue
					}
					nonemptyLines = append(nonemptyLines, l)
				}
				count := len(nonemptyLines)
				if count != c.wantCount {
					t.Errorf("filter %q got %d results, want %d; output: %s",
						c.filters, count, c.wantCount, got)
				}
			})
		}
	})
}
