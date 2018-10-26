//+build smoke

package smoke

import "testing"

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

		client.MustRun(t, "manifest get", flags.ManifestIDFlags())
	})
}
