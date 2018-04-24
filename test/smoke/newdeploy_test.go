//+build smoke

package smoke

import (
	"strings"
	"testing"

	sous "github.com/opentable/sous/lib"
)

const simpleServer = `
FROM alpine
CMD if [ -z "$T" ]; then T=2; fi; echo -n "Sleeping ${T}s..."; sleep $T; echo "Done"; echo "Listening on :$PORT0"; while true; do echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0; done`

// setupProject creates a brand new git repo containing the provided Dockerfile,
// commits that Dockerfile, runs 'sous version' and 'sous config', and returns a
// sous TestClient in the project directory.
func setupProject(t *testing.T, f TestFixture, dockerfile string) TestClient {
	t.Helper()
	// Setup project git repo.
	projectDir := makeGitRepo(t, f.BaseDir, "projects/project1", GitRepoSpec{
		UserName:  "Sous User 1",
		UserEmail: "sous-user1@example.com",
		OriginURL: "git@github.com:user1/repo1.git",
	})
	makeFileString(t, projectDir, "Dockerfile", dockerfile)
	mustDoCMD(t, projectDir, "git", "add", "Dockerfile")
	mustDoCMD(t, projectDir, "git", "commit", "-m", "Add Dockerfile")

	sous := f.Client

	// Dump sous version & config.
	t.Logf("Sous version: %s", sous.MustRun(t, "version", nil))
	sous.MustRun(t, "config", nil)

	// cd into project dir
	sous.Dir = projectDir

	return sous
}

type sousFlags struct {
	kind    string
	flavor  string
	cluster string
	repo    string
	offset  string
	tag     string
}

func (f *sousFlags) Args() []string {
	if f == nil {
		return nil
	}
	var out []string
	if f.kind != "" {
		out = append(out, "-kind", f.kind)
	}
	if f.flavor != "" {
		out = append(out, "-flavor", f.flavor)
	}
	if f.cluster != "" {
		out = append(out, "-cluster", f.cluster)
	}
	if f.repo != "" {
		out = append(out, "-repo", f.repo)
	}
	if f.offset != "" {
		out = append(out, "-offset", f.offset)
	}
	if f.tag != "" {
		out = append(out, "-tag", f.tag)
	}
	return out
}

func defaultManifestID() sous.ManifestID {
	return sous.ManifestID{
		Source: sous.SourceLocation{
			Dir:  "",
			Repo: "github.com/user1/repo1",
		},
		Flavor: "",
	}
}

func manifestID(repo, dir, flavor string) sous.ManifestID {
	return sous.ManifestID{
		Source: sous.SourceLocation{
			Dir:  dir,
			Repo: repo,
		},
		Flavor: flavor,
	}
}

func deploymentID(mid sous.ManifestID, cluster string) sous.DeploymentID {
	return sous.DeploymentID{
		ManifestID: mid,
		Cluster:    cluster,
	}
}

func defaultDeploymentID() sous.DeploymentID {
	return sous.DeploymentID{
		ManifestID: defaultManifestID(),
		Cluster:    "cluster1",
	}
}

func TestSousNewdeploy(t *testing.T) {

	t.Run("simple", func(t *testing.T) {
		f := newTestFixture(t)
		sous := setupProject(t, f, simpleServer)
		sous.MustRun(t, "init", nil, "-kind", "http-service")
		sous.TransformManifestAsString(t, nil, func(manifest string) string {
			return strings.Replace(manifest, "NumInstances: 0", "NumInstances: 1", -1)
		})
		sous.MustRun(t, "build", nil, "-tag", "1.2.3")
		sous.MustRun(t, "newdeploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")
	})

	t.Run("flavors", func(t *testing.T) {
		f := newTestFixture(t)
		sous := setupProject(t, f, simpleServer)
		flavor := "flavor1"
		flavorFlag := &sousFlags{flavor: flavor}
		sous.MustRun(t, "init", flavorFlag, "-kind", "http-service")
		sous.TransformManifestAsString(t, flavorFlag, func(manifest string) string {
			return strings.Replace(manifest, "NumInstances: 0", "NumInstances: 1", -1)
		})
		sous.MustRun(t, "build", nil, "-tag", "1.2.3")
		sous.MustRun(t, "newdeploy", flavorFlag, "-cluster", "cluster1", "-tag", "1.2.3")
	})

	t.Run("pause-unpause", func(t *testing.T) {
		f := newTestFixture(t)
		sous := setupProject(t, f, simpleServer)
		sous.MustRun(t, "init", nil, "-kind", "http-service")
		sous.TransformManifestAsString(t, nil, func(manifest string) string {
			return strings.Replace(manifest, "NumInstances: 0", "NumInstances: 1", -1)
		})
		sous.MustRun(t, "build", nil, "-tag", "1")
		sous.MustRun(t, "build", nil, "-tag", "2")
		sous.MustRun(t, "build", nil, "-tag", "3")
		sous.MustRun(t, "newdeploy", nil, "-cluster", "cluster1", "-tag", "1")
		f.Singularity.PauseRequestForDeployment(t, deploymentID(defaultManifestID(), "cluster1"))
		sous.MustFail(t, "newdeploy", nil, "-cluster", "cluster1", "-tag", "2")
		f.Singularity.UnpauseRequestForDeployment(t, deploymentID(defaultManifestID(), "cluster1"))
		knownToFailHere(t)
		sous.MustRun(t, "newdeploy", nil, "-cluster", "cluster1", "-tag", "3")
	})
}
