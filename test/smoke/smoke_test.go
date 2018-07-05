//+build smoke

package smoke

import (
	"flag"
	"fmt"
	"os"
	"testing"

	sous "github.com/opentable/sous/lib"
)

// Define some Dockerfiles for use in tests.
const (
	simpleServer = `
FROM alpine
CMD if [ -z "$T" ]; then T=2; fi; echo -n "Sleeping ${T}s..."; sleep $T; echo "Done"; echo "Listening on :$PORT0"; while true; do echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0; done
`
	sleeper = `
FROM alpine
CMD echo -n Sleeping for 10s...; sleep 10; echo Done
`
	failer = `
FROM alpine
CMD echo -n Failing in 10s...; sleep 10; echo Failed; exit 1
`
)

// setupProject creates a brand new git repo containing the provided Dockerfile,
// commits that Dockerfile, runs 'sous version' and 'sous config', and returns a
// sous TestClient in the project directory.
func setupProject(t *testing.T, f *TestFixture, dockerfile string) *TestClient {
	t.Helper()
	// Setup project git repo.
	projectDir := makeGitRepo(t, f.Client.BaseDir, "projects/project1", GitRepoSpec{
		UserName:  "Sous User 1",
		UserEmail: "sous-user1@example.com",
		OriginURL: "git@github.com:user1/repo1.git",
	})
	makeFileString(t, projectDir, "Dockerfile", dockerfile)
	mustDoCMD(t, projectDir, "git", "add", "Dockerfile")
	mustDoCMD(t, projectDir, "git", "commit", "-m", "Add Dockerfile")

	client := f.Client

	// cd into project dir
	client.Dir = projectDir

	// Dump sous version & config.
	t.Logf("Sous version: %s", client.MustRun(t, "version", nil))
	client.MustRun(t, "config", nil)

	return client
}

var pf *ParallelTestFixture

func TestMain(m *testing.M) {
	flag.Parse()
	exitCode := m.Run()
	pf.PrintSummary()
	os.Exit(exitCode)
}

func Test(t *testing.T) {
	t.Parallel()

	fixtureConfigs := []fixtureConfig{
		{dbPrimary: false},
		{dbPrimary: true},
	}

	// Set the global parallel test fixture.
	pf = newParallelTestFixture(t, PTFOpts{
		NumFreeAddrs: 128,
	})

	for _, fixtureConfig := range fixtureConfigs {
		t.Run(fixtureConfig.Desc(), func(t *testing.T) {
			t.Parallel()
			t.Run("simple", func(t *testing.T) {
				f := pf.NewIsolatedFixture(t, fixtureConfig)
				defer f.ReportStatus(t)
				client := setupProject(t, f, simpleServer)
				client.MustRun(t, "init", nil, "-kind", "http-service")
				client.TransformManifest(t, nil, setMinimalMemAndCPUNumInst1)
				client.MustRun(t, "build", nil, "-tag", "1.2.3")
				client.MustRun(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")

				did := sous.DeploymentID{
					ManifestID: sous.ManifestID{
						Source: sous.SourceLocation{
							Repo: "github.com/user1/repo1",
						},
					},
					Cluster: "cluster1",
				}

				reqID := f.Singularity.DefaultReqID(t, did)
				assertActiveStatus(t, f, reqID)
				assertSingularityRequestTypeService(t, f, reqID)
				assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
			})

			t.Run("zero-instances", func(t *testing.T) {
				f := pf.NewIsolatedFixture(t, fixtureConfig)
				defer f.ReportStatus(t)
				client := setupProject(t, f, simpleServer)
				client.MustRun(t, "init", nil, "-kind", "http-service")
				client.TransformManifest(t, nil, setMinimalMemAndCPUNumInst0)
				client.MustRun(t, "build", nil, "-tag", "1.2.3")

				client.MustFail(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")
			})

			t.Run("fails", func(t *testing.T) {
				f := pf.NewIsolatedFixture(t, fixtureConfig)
				defer f.ReportStatus(t)
				client := setupProject(t, f, failer)
				client.MustRun(t, "init", nil, "-kind", "http-service")
				client.TransformManifest(t, nil, setMinimalMemAndCPUNumInst1)
				client.MustRun(t, "build", nil, "-tag", "1.2.3")
				client.MustFail(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")

				did := sous.DeploymentID{
					ManifestID: sous.ManifestID{
						Source: sous.SourceLocation{
							Repo: "github.com/user1/repo1",
						},
					},
					Cluster: "cluster1",
				}

				reqID := f.Singularity.DefaultReqID(t, did)
				assertActiveStatus(t, f, reqID)
				assertSingularityRequestTypeService(t, f, reqID)
				assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
			})

			t.Run("flavors", func(t *testing.T) {
				f := pf.NewIsolatedFixture(t, fixtureConfig)
				defer f.ReportStatus(t)
				client := setupProject(t, f, simpleServer)
				flavor := "flavor1"
				flavorFlag := &sousFlags{flavor: flavor}
				client.MustRun(t, "init", flavorFlag, "-kind", "http-service")
				client.TransformManifest(t, flavorFlag, setMinimalMemAndCPUNumInst1)
				client.MustRun(t, "build", nil, "-tag", "1.2.3")
				client.MustRun(t, "deploy", flavorFlag, "-cluster", "cluster1", "-tag", "1.2.3")

				did := sous.DeploymentID{
					ManifestID: sous.ManifestID{
						Source: sous.SourceLocation{
							Repo: "github.com/user1/repo1",
						},
						Flavor: flavor,
					},
					Cluster: "cluster1",
				}

				reqID := f.Singularity.DefaultReqID(t, did)
				assertActiveStatus(t, f, reqID)
				assertSingularityRequestTypeService(t, f, reqID)
				assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
			})

			t.Run("pause-unpause", func(t *testing.T) {
				f := pf.NewIsolatedFixture(t, fixtureConfig)
				defer f.ReportStatus(t)
				client := setupProject(t, f, simpleServer)
				client.MustRun(t, "init", nil, "-kind", "http-service")
				client.TransformManifest(t, nil, setMinimalMemAndCPUNumInst1)
				client.MustRun(t, "build", nil, "-tag", "1")
				client.MustRun(t, "build", nil, "-tag", "2")
				client.MustRun(t, "build", nil, "-tag", "3")
				client.MustRun(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1")

				did := sous.DeploymentID{
					ManifestID: sous.ManifestID{
						Source: sous.SourceLocation{
							Repo: "github.com/user1/repo1",
						},
					},
					Cluster: "cluster1",
				}
				reqID := f.Singularity.DefaultReqID(t, did)
				assertActiveStatus(t, f, reqID)
				assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
				assertSingularityRequestTypeService(t, f, reqID)

				f.Singularity.PauseRequestForDeployment(t, reqID)
				client.MustFail(t, "deploy", nil, "-cluster", "cluster1", "-tag", "2")
				f.Singularity.UnpauseRequestForDeployment(t, reqID)

				client.MustRun(t, "deploy", nil, "-cluster", "cluster1", "-tag", "3")
				assertActiveStatus(t, f, reqID)
			})

			t.Run("scheduled", func(t *testing.T) {
				f := pf.NewIsolatedFixture(t, fixtureConfig)
				defer f.ReportStatus(t)
				client := setupProject(t, f, sleeper)
				client.MustRun(t, "init", nil, "-kind", "scheduled")
				client.TransformManifest(t, nil, func(m sous.Manifest) sous.Manifest {
					clusterName := "cluster1" + f.ClusterSuffix
					d := m.Deployments[clusterName]
					d.NumInstances = 1
					d.Schedule = "*/5 * * * *"
					m.Deployments[clusterName] = d

					m.Deployments = setMemAndCPUForAll(m.Deployments)

					return m
				})
				client.MustRun(t, "build", nil, "-tag", "1.2.3")
				client.MustRun(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")

				did := sous.DeploymentID{
					ManifestID: sous.ManifestID{
						Source: sous.SourceLocation{
							Repo: "github.com/user1/repo1",
						},
					},
					Cluster: "cluster1",
				}

				reqID := f.Singularity.DefaultReqID(t, did)
				assertSingularityRequestTypeScheduled(t, f, reqID)
				assertActiveStatus(t, f, reqID)
				assertNilHealthCheckOnLatestDeploy(t, f, reqID)
			})

			t.Run("custom-reqid-first-deploy", func(t *testing.T) {
				f := pf.NewIsolatedFixture(t, fixtureConfig)
				defer f.ReportStatus(t)
				client := setupProject(t, f, simpleServer)
				client.MustRun(t, "init", nil, "-kind", "http-service")
				client.MustRun(t, "build", nil, "-tag", "1.2.3")

				customID := "some-custom-req-id" + f.ClusterSuffix
				client.SetSingularityRequestID(t, nil, "cluster1", customID)

				client.MustRun(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")

				assertSingularityRequestTypeService(t, f, customID)
				assertActiveStatus(t, f, customID)
				assertNonNilHealthCheckOnLatestDeploy(t, f, customID)

			})

			t.Run("custom-reqid-second-deploy", func(t *testing.T) {
				f := pf.NewIsolatedFixture(t, fixtureConfig)
				defer f.ReportStatus(t)
				client := setupProject(t, f, simpleServer)
				client.MustRun(t, "init", nil, "-kind", "http-service")
				client.MustRun(t, "build", nil, "-tag", "1.2.3")

				client.MustRun(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")

				did := sous.DeploymentID{
					ManifestID: sous.ManifestID{
						Source: sous.SourceLocation{
							Repo: "github.com/user1/repo1",
						},
					},
					Cluster: "cluster1",
				}
				originalReqID := f.Singularity.DefaultReqID(t, did)
				assertSingularityRequestTypeService(t, f, originalReqID)
				assertActiveStatus(t, f, originalReqID)
				assertNonNilHealthCheckOnLatestDeploy(t, f, originalReqID)

				customID := "some-custom-req-id" + f.ClusterSuffix
				client.SetSingularityRequestID(t, nil, "cluster1", customID)

				// Force to avoid having to make another build.
				client.MustRun(t, "deploy", nil, "-force", "-cluster", "cluster1", "-tag", "1.2.3")

				assertSingularityRequestTypeService(t, f, customID)
				assertActiveStatus(t, f, customID)
				assertNonNilHealthCheckOnLatestDeploy(t, f, customID)

				// TODO: Implement cleanup of old request.
				//assertRequestDoesNotExist(t, f, originalReqID)

				assertActiveStatus(t, f, originalReqID)
			})

			t.Run("change-reqid", func(t *testing.T) {
				f := pf.NewIsolatedFixture(t, fixtureConfig)
				defer f.ReportStatus(t)
				client := setupProject(t, f, simpleServer)
				client.MustRun(t, "init", nil, "-kind", "http-service")
				client.MustRun(t, "build", nil, "-tag", "1.2.3")

				customID1 := "some-custom-req-id1" + f.ClusterSuffix
				customID2 := "some-custom-req-id2" + f.ClusterSuffix

				client.SetSingularityRequestID(t, nil, "cluster1", customID1)
				client.MustRun(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")

				assertSingularityRequestTypeService(t, f, customID1)
				assertActiveStatus(t, f, customID1)
				assertNonNilHealthCheckOnLatestDeploy(t, f, customID1)

				client.SetSingularityRequestID(t, nil, "cluster1", customID2)
				client.MustRun(t, "deploy", nil, "-force", "-cluster", "cluster1", "-tag", "1.2.3")

				assertSingularityRequestTypeService(t, f, customID2)
				assertActiveStatus(t, f, customID2)
				assertNonNilHealthCheckOnLatestDeploy(t, f, customID2)

				// TODO: Implement cleanup of old request.
				//assertRequestDoesNotExist(t, f, customID1)

				assertActiveStatus(t, f, customID1)
			})

			t.Run("add-artifact", func(t *testing.T) {
				f := pf.NewIsolatedFixture(t, fixtureConfig)
				defer f.ReportStatus(t)
				client := setupProject(t, f, simpleServer)

				reg := f.Client.Config.Docker.RegistryHost
				repo := "github.com/user1/project1"
				tag := "1.2.3"
				dockerTag := f.IsolatedVersionTag(t, tag)
				dockerRepo := fmt.Sprintf("%s/%s", reg, repo)
				dockerRef := fmt.Sprintf("%s:%s", dockerRepo, dockerTag)
				mustDoCMD(t, client.Dir, "docker", "build", "-t", dockerRef, ".")
				mustDoCMD(t, client.Dir, "docker", "push", dockerRef)

				client.MustRun(t, "add artifact", nil, "-docker-image", dockerRepo, "-repo", repo, "-tag", tag)
				// TODO: Assert artifact stored correctly.
			})
		})
	}
}
