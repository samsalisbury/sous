//+build smoke

package smoke

import (
	"strings"
	"testing"
)

// TestBugs is a place to reproduce bug reports from users for fixing.
// They may later be migrated elsewhere.
func TestBugs(t *testing.T) {
	m := newRunner(t, matrix())

	m.Run("manifest-get-flavor-offset-bug", func(t *testing.T, f *fixture) {
		client := setupProject(t, f,
			f.Projects.HTTPServer().PrefixAll("src/stage1/whosonfirst-gb-postcodes"))

		flags := &sousFlags{kind: "http-service",
			tag:     "1.2.3",
			cluster: "cluster1",
			repo:    "github.com/user1/repo1",
			offset:  "src/stage1/whosonfirst-gb-postcodes",
			flavor:  "",
		}

		initProject(t, client, flags, setMinimalMemAndCPUNumInst1)

		// Point flags at nonexistent manifest...
		flags.flavor = "src/stage1"

		// On writing this test, the error is reproduced:
		// > No manifest matched by <cluster:* repo:github.com/user1/repo1 offset:* flavor:src/stage1 tag:src/stage1/whosonfirst-gb-postcodes revision:*>
		// Note that 'tag' and 'offset' are reversed in the above string.

		got := client.MustFail(t, "manifest get", flags.ManifestIDFlags())
		want := `No manifest matched by <cluster:* repo:github.com/user1/repo1 offset:src/stage1/whosonfirst-gb-postcodes flavor:src/stage1 tag:* revision:*>`
		if !strings.Contains(got, want) {
			t.Errorf("got stderr %q; want it to contain %q", got, want)
		}
	})

	m.Run("uppercase-docker-repo-bug", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer(), func(p *sousProjectConfig) {
			p.gitRepoSpec.OriginURL = "git@github.com:SomeUser/SomeProject.git"
		})

		flags := &sousFlags{
			repo: "github.com/SomeUser/SomeProject",
			tag:  "1.2.3",
		}

		// On writing this test, it fails because it tries to create a docker
		// image ref with upper-case characters in its repo component, which
		// is not allowed:
		//   shell> docker build -t 192.168.99.100:5000/SomeUser/SomeProject:1.2.3-testbugs-git-simple-uppercase-docker-repo-bug -t 192.168.99.100:5000/SomeUser/SomeProject:z-2018-10-29T13.12.21 -
		//   invalid argument "192.168.99.100:5000/SomeUser/SomeProject:1.2.3-testbugs-git-simple-uppercase-docker-repo-bug" for "-t, --tag" flag: invalid reference format: repository name must be lowercase

		p.MustRun(t, "build", flags)
	})
}
