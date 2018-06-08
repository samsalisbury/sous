//+build smoke

package smoke

import (
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
func setupProject(t *testing.T, f TestFixture, dockerfile string) *TestClient {
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
	client.ClusterSuffix = f.ClusterSuffix

	// Dump sous version & config.
	t.Logf("Sous version: %s", client.MustRun(t, "version", nil))
	client.MustRun(t, "config", nil)

	return client
}

func TestSousDeploy(t *testing.T) {

	fixtureConfigs := []fixtureConfig{
		{dbPrimary: false},
		{dbPrimary: true},
	}

	pf := newParallelTestFixture(t, PTFOpts{
		NumFreeAddrs: 128,
	})

	// This outer Run call does not return until all inner parallel tests have
	// finished. This allows the final pf.PrintSummary call to happen only after
	// all tests have concluded.
	t.Run("_", func(t *testing.T) {

		for _, fixtureConfig := range fixtureConfigs {
			t.Run(fixtureConfig.Desc(), func(t *testing.T) {
				t.Parallel()
				deployCommand := "deploy"
				t.Run(deployCommand, func(t *testing.T) {
					t.Parallel()

					t.Run("simple", func(t *testing.T) {
						f := pf.NewIsolatedFixture(t, fixtureConfig)
						defer f.ReportStatus(t)
						client := setupProject(t, f, simpleServer)
						client.MustRun(t, "init", nil, "-kind", "http-service")
						client.TransformManifest(t, nil, setMinimalMemAndCPUNumInst1)
						client.MustRun(t, "build", nil, "-tag", "1.2.3")
						client.MustRun(t, deployCommand, nil, "-cluster", "cluster1", "-tag", "1.2.3")

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

					t.Run("zero-instances", func(t *testing.T) {
						f := pf.NewIsolatedFixture(t, fixtureConfig)
						defer f.ReportStatus(t)
						client := setupProject(t, f, simpleServer)
						client.MustRun(t, "init", nil, "-kind", "http-service")
						client.TransformManifest(t, nil, setMinimalMemAndCPUNumInst0)
						client.MustRun(t, "build", nil, "-tag", "1.2.3")

						client.MustFail(t, deployCommand, nil, "-cluster", "cluster1", "-tag", "1.2.3")
					})

					t.Run("fails", func(t *testing.T) {
						f := pf.NewIsolatedFixture(t, fixtureConfig)
						defer f.ReportStatus(t)
						client := setupProject(t, f, failer)
						client.MustRun(t, "init", nil, "-kind", "http-service")
						client.TransformManifest(t, nil, setMinimalMemAndCPUNumInst1)
						client.MustRun(t, "build", nil, "-tag", "1.2.3")
						client.MustFail(t, deployCommand, nil, "-cluster", "cluster1", "-tag", "1.2.3")

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
						f := pf.NewIsolatedFixture(t, fixtureConfig)
						defer f.ReportStatus(t)
						client := setupProject(t, f, simpleServer)
						flavor := "flavor1"
						flavorFlag := &sousFlags{flavor: flavor}
						client.MustRun(t, "init", flavorFlag, "-kind", "http-service")
						client.TransformManifest(t, flavorFlag, setMinimalMemAndCPUNumInst1)
						client.MustRun(t, "build", nil, "-tag", "1.2.3")
						client.MustRun(t, deployCommand, flavorFlag, "-cluster", "cluster1", "-tag", "1.2.3")

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
						f := pf.NewIsolatedFixture(t, fixtureConfig)
						defer f.ReportStatus(t)
						client := setupProject(t, f, simpleServer)
						client.MustRun(t, "init", nil, "-kind", "http-service")
						client.TransformManifest(t, nil, setMinimalMemAndCPUNumInst1)
						client.MustRun(t, "build", nil, "-tag", "1")
						client.MustRun(t, "build", nil, "-tag", "2")
						client.MustRun(t, "build", nil, "-tag", "3")
						client.MustRun(t, deployCommand, nil, "-cluster", "cluster1", "-tag", "1")

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
						client.MustFail(t, deployCommand, nil, "-cluster", "cluster1", "-tag", "2")
						f.Singularity.UnpauseRequestForDeployment(t, did)

						f.KnownToFailHere(t)

						client.MustRun(t, deployCommand, nil, "-cluster", "cluster1", "-tag", "3")
						assertActiveStatus(t, f, did)
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
						client.MustRun(t, deployCommand, nil, "-cluster", "cluster1", "-tag", "1.2.3")

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
				})
			})
		}
	})

	pf.PrintSummary(t)
}
