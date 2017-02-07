// +build integration

package integration

import (
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/filemap"
)

func TestSingularityGetRunningDeploymentSet(t *testing.T) {
	// Define repos.
	type TestRepo struct {
		Source sous.SourceLocation
		Files  filemap.FileMap
	}

	repos := []TestRepo{
		Source: sous.ParseSourceLocation("github.com/user/project1"),
		filemap.FileMap{
			"Dockerfile": `
			FROM busybox
			CMD echo "Done"
			`,
		},
	}
}
