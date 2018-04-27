//+build smoke

package smoke

import (
	"strings"
	"testing"

	"github.com/opentable/go-singularity/dtos"
	sous "github.com/opentable/sous/lib"
)

const simpleServer = `
FROM alpine
CMD if [ -z "$T" ]; then T=2; fi; echo -n "Sleeping ${T}s..."; sleep $T; echo "Done"; echo "Listening on :$PORT0"; while true; do echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0; done
`

const sleeper = `
FROM alpine
CMD echo -n Sleeping for 10s...; sleep 10; echo Done
`

const failer = `
FROM alpine
CMD echo -n Failing in 10s...; sleep 10; echo Failed; exit 1
`

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

	client := f.Client

	// Dump sous version & config.
	t.Logf("Sous version: %s", client.MustRun(t, "version", nil))
	client.MustRun(t, "config", nil)

	// cd into project dir
	client.Dir = projectDir

	return client
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

func assertActiveStatus(t *testing.T, f TestFixture, did sous.DeploymentID) {
	req := f.Singularity.GetRequestForDeployment(t, did)
	gotStatus := req.State
	wantStatus := dtos.SingularityRequestParentRequestStateACTIVE
	if gotStatus != wantStatus {
		t.Fatalf("got status %v; want %v", gotStatus, wantStatus)
	}
}

func TestSousNewdeploy(t *testing.T) {

	t.Run("simple", func(t *testing.T) {
		f := newTestFixture(t)
		client := setupProject(t, f, simpleServer)
		client.MustRun(t, "init", nil, "-kind", "http-service")
		client.TransformManifestAsString(t, nil, func(manifest string) string {
			return strings.Replace(manifest, "NumInstances: 0", "NumInstances: 1", -1)
		})
		client.MustRun(t, "build", nil, "-tag", "1.2.3")
		client.MustRun(t, "newdeploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")

		did := sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user1/repo1",
				},
			},
			Cluster: "cluster1",
		}

		assertActiveStatus(t, f, did)
		assertSingularityRequestTypeService(t, f, did)
		assertNonNilHealthCheckOnLatestDeploy(t, f, did)
	})

	t.Run("fails", func(t *testing.T) {
		f := newTestFixture(t)
		client := setupProject(t, f, failer)
		client.MustRun(t, "init", nil, "-kind", "http-service")
		client.TransformManifestAsString(t, nil, func(manifest string) string {
			return strings.Replace(manifest, "NumInstances: 0", "NumInstances: 1", -1)
		})
		client.MustRun(t, "build", nil, "-tag", "1.2.3")
		client.MustFail(t, "newdeploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")

		did := sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user1/repo1",
				},
			},
			Cluster: "cluster1",
		}

		assertActiveStatus(t, f, did)
		assertSingularityRequestTypeService(t, f, did)
		assertNonNilHealthCheckOnLatestDeploy(t, f, did)
	})

	t.Run("flavors", func(t *testing.T) {
		f := newTestFixture(t)
		client := setupProject(t, f, simpleServer)
		flavor := "flavor1"
		flavorFlag := &sousFlags{flavor: flavor}
		client.MustRun(t, "init", flavorFlag, "-kind", "http-service")
		client.TransformManifestAsString(t, flavorFlag, func(manifest string) string {
			return strings.Replace(manifest, "NumInstances: 0", "NumInstances: 1", -1)
		})
		client.MustRun(t, "build", nil, "-tag", "1.2.3")
		client.MustRun(t, "newdeploy", flavorFlag, "-cluster", "cluster1", "-tag", "1.2.3")

		did := sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user1/repo1",
				},
				Flavor: flavor,
			},
			Cluster: "cluster1",
		}

		assertActiveStatus(t, f, did)
		assertSingularityRequestTypeService(t, f, did)
		assertNonNilHealthCheckOnLatestDeploy(t, f, did)
	})

	t.Run("pause-unpause", func(t *testing.T) {
		f := newTestFixture(t)
		client := setupProject(t, f, simpleServer)
		client.MustRun(t, "init", nil, "-kind", "http-service")
		client.TransformManifestAsString(t, nil, func(manifest string) string {
			return strings.Replace(manifest, "NumInstances: 0", "NumInstances: 1", -1)
		})
		client.MustRun(t, "build", nil, "-tag", "1")
		client.MustRun(t, "build", nil, "-tag", "2")
		client.MustRun(t, "build", nil, "-tag", "3")
		client.MustRun(t, "newdeploy", nil, "-cluster", "cluster1", "-tag", "1")

		did := sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user1/repo1",
				},
			},
			Cluster: "cluster1",
		}
		assertActiveStatus(t, f, did)
		assertNonNilHealthCheckOnLatestDeploy(t, f, did)
		assertSingularityRequestTypeService(t, f, did)

		f.Singularity.PauseRequestForDeployment(t, did)
		client.MustFail(t, "newdeploy", nil, "-cluster", "cluster1", "-tag", "2")
		f.Singularity.UnpauseRequestForDeployment(t, did)

		knownToFailHere(t)

		client.MustRun(t, "newdeploy", nil, "-cluster", "cluster1", "-tag", "3")
		assertActiveStatus(t, f, did)
	})

	t.Run("scheduled", func(t *testing.T) {
		f := newTestFixture(t)
		client := setupProject(t, f, sleeper)
		client.MustRun(t, "init", nil, "-kind", "scheduled")
		client.TransformManifest(t, nil, func(m sous.Manifest) sous.Manifest {
			d := m.Deployments["cluster1"]
			d.NumInstances = 1
			d.Schedule = "*/5 * * * *"
			m.Deployments["cluster1"] = d
			return m
		})
		client.MustRun(t, "build", nil, "-tag", "1.2.3")
		client.MustRun(t, "newdeploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")

		did := sous.DeploymentID{
			ManifestID: sous.ManifestID{
				Source: sous.SourceLocation{
					Repo: "github.com/user1/repo1",
				},
			},
			Cluster: "cluster1",
		}

		assertSingularityRequestTypeScheduled(t, f, did)
		assertActiveStatus(t, f, did)
		assertNilHealthCheckOnLatestDeploy(t, f, did)
	})
}

func assertSingularityRequestTypeScheduled(t *testing.T, f TestFixture, did sous.DeploymentID) {
	t.Helper()
	req := f.Singularity.GetRequestForDeployment(t, did)
	gotType := req.Request.RequestType
	wantType := dtos.SingularityRequestRequestTypeSCHEDULED
	if gotType != wantType {
		t.Errorf("got request type %v; want %v", gotType, wantType)
	}
}

func assertSingularityRequestTypeService(t *testing.T, f TestFixture, did sous.DeploymentID) {
	t.Helper()
	req := f.Singularity.GetRequestForDeployment(t, did)
	gotType := req.Request.RequestType
	wantType := dtos.SingularityRequestRequestTypeSERVICE
	if gotType != wantType {
		t.Errorf("got request type %v; want %v", gotType, wantType)
	}
}

func assertNilHealthCheckOnLatestDeploy(t *testing.T, f TestFixture, did sous.DeploymentID) {
	t.Helper()
	dep := f.Singularity.GetLatestDeployForDeployment(t, did)
	gotHealthcheck := dep.Deploy.Healthcheck
	if gotHealthcheck != nil {
		t.Fatalf("got Healthcheck = %v; want nil", gotHealthcheck)
	}
}

func assertNonNilHealthCheckOnLatestDeploy(t *testing.T, f TestFixture, did sous.DeploymentID) {
	t.Helper()
	dep := f.Singularity.GetLatestDeployForDeployment(t, did)
	gotHealthcheck := dep.Deploy.Healthcheck
	if gotHealthcheck == nil {
		t.Fatalf("got nil Healthcheck")
	}
}
