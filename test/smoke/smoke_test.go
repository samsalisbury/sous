//+build smoke

package smoke

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

// initBuild is a macro to fast-forward to tests to a point where we have
// initialised the project, transformed the manifest, and performed a build.
func initBuild(t *testing.T, client *TestClient, flags *sousFlags, transforms ...ManifestTransform) {
	client.MustRun(t, "init", flags.SousInitFlags())
	client.TransformManifest(t, flags.ManifestIDFlags(), transforms...)
	client.MustRun(t, "build", flags.SourceIDFlags())
}

// initBuildDeploy is a macro for fast-forwarding tests to a point where we have
// initialised the project as a kind project, built it with -tag tag and
// deployed that tag successfully to cluster.
func initBuildDeploy(t *testing.T, client *TestClient, flags *sousFlags, transforms ...ManifestTransform) {
	initBuild(t, client, flags, transforms...)
	client.MustRun(t, "deploy", flags.SousDeployFlags())
}

func TestInitToDeploy(t *testing.T) {
	pf := pfs.newParallelTestFixture(t, Matrix())

	pf.RunMatrix(

		PTest{Name: "simple", Test: func(t *testing.T, f *TestFixture) {
			_, reqID := f.DIDAndDefaultReqID(t, "github.com/user1/repo1", "", "", "cluster1")

			client := f.setupProject(t, f.Projects.HTTPServer())

			flags := &sousFlags{kind: "http-service", tag: "1.2.3", cluster: "cluster1"}

			initBuildDeploy(t, client, flags, setMinimalMemAndCPUNumInst1)

			assertActiveStatus(t, f, reqID)
			assertSingularityRequestTypeService(t, f, reqID)
			assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
		}},

		PTest{Name: "fail-zero-instances", Test: func(t *testing.T, f *TestFixture) {
			client := f.setupProject(t, f.Projects.HTTPServer())

			flags := &sousFlags{kind: "http-service", tag: "1.2.3"}

			initBuild(t, client, flags, setMinimalMemAndCPUNumInst0)

			client.MustFail(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")
		}},

		PTest{Name: "fail-container-crash", Test: func(t *testing.T, f *TestFixture) {
			client := f.setupProject(t, f.Projects.Failer())

			flags := &sousFlags{kind: "http-service", tag: "1.2.3"}

			initBuild(t, client, flags, setMinimalMemAndCPUNumInst1)

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
		}},

		PTest{Name: "flavors", Test: func(t *testing.T, f *TestFixture) {
			client := f.setupProject(t, f.Projects.HTTPServer())

			flags := &sousFlags{
				kind: "http-service", tag: "1.2.3", cluster: "cluster1",
				flavor: "flavor1",
			}

			initBuildDeploy(t, client, flags, setMinimalMemAndCPUNumInst1)

			did := sous.DeploymentID{
				ManifestID: sous.ManifestID{
					Source: sous.SourceLocation{
						Repo: "github.com/user1/repo1",
					},
					Flavor: "flavor1",
				},
				Cluster: "cluster1",
			}

			reqID := f.Singularity.DefaultReqID(t, did)
			assertActiveStatus(t, f, reqID)
			assertSingularityRequestTypeService(t, f, reqID)
			assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
		}},

		PTest{Name: "pause-unpause", Test: func(t *testing.T, f *TestFixture) {
			client := f.setupProject(t, f.Projects.HTTPServer())

			flags := &sousFlags{kind: "http-service", tag: "1", cluster: "cluster1"}

			initBuildDeploy(t, client, flags, setMinimalMemAndCPUNumInst1)

			// Prepare a couple more builds...
			client.MustRun(t, "build", nil, "-tag", "2")
			client.MustRun(t, "build", nil, "-tag", "3")

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
		}},

		PTest{Name: "scheduled", Test: func(t *testing.T, f *TestFixture) {
			client := f.setupProject(t, f.Projects.Sleeper())

			flags := &sousFlags{kind: "scheduled", tag: "1.2.3", cluster: "cluster1"}

			initBuildDeploy(t, client, flags, func(m sous.Manifest) sous.Manifest {
				clusterName := "cluster1" + f.ClusterSuffix
				d := m.Deployments[clusterName]
				d.NumInstances = 1
				d.Schedule = "*/5 * * * *"
				m.Deployments[clusterName] = d

				m.Deployments = setMemAndCPUForAll(m.Deployments)

				return m
			})

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
		}},

		PTest{Name: "custom-reqid-first-deploy", Test: func(t *testing.T, f *TestFixture) {
			client := f.setupProject(t, f.Projects.HTTPServer())

			flags := &sousFlags{kind: "http-service", tag: "1.2.3", cluster: "cluster1"}

			customID := "some-custom-req-id" + f.ClusterSuffix

			initBuildDeploy(t, client, flags,
				setMinimalMemAndCPUNumInst1,
				client.setSingularityRequestID(t, "cluster1", customID),
			)

			assertSingularityRequestTypeService(t, f, customID)
			assertActiveStatus(t, f, customID)
			assertNonNilHealthCheckOnLatestDeploy(t, f, customID)
		}},

		PTest{Name: "custom-reqid-second-deploy", Test: func(t *testing.T, f *TestFixture) {
			client := f.setupProject(t, f.Projects.HTTPServer())

			flags := &sousFlags{kind: "http-service", tag: "1.2.3", cluster: "cluster1"}

			initBuildDeploy(t, client, flags, setMinimalMemAndCPUNumInst1)

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
			client.TransformManifest(t, nil, client.setSingularityRequestID(t, "cluster1", customID))

			// Force to avoid having to make another build.
			client.MustRun(t, "deploy", nil, "-force", "-cluster", "cluster1", "-tag", "1.2.3")

			assertSingularityRequestTypeService(t, f, customID)
			assertActiveStatus(t, f, customID)
			assertNonNilHealthCheckOnLatestDeploy(t, f, customID)

			// TODO: Implement cleanup of old request.
			//assertRequestDoesNotExist(t, f, originalReqID)

			assertActiveStatus(t, f, originalReqID)
		}},

		PTest{Name: "change-reqid", Test: func(t *testing.T, f *TestFixture) {
			client := f.setupProject(t, f.Projects.HTTPServer())

			flags := &sousFlags{kind: "http-service", tag: "1.2.3", cluster: "cluster1"}

			customID1 := "some-custom-req-id1" + f.ClusterSuffix

			initBuildDeploy(t, client, flags,
				setMinimalMemAndCPUNumInst1,
				client.setSingularityRequestID(t, "cluster1", customID1),
			)

			assertSingularityRequestTypeService(t, f, customID1)
			assertActiveStatus(t, f, customID1)
			assertNonNilHealthCheckOnLatestDeploy(t, f, customID1)

			customID2 := "some-custom-req-id2" + f.ClusterSuffix

			client.TransformManifest(t, nil, client.setSingularityRequestID(t, "cluster1", customID2))

			client.MustRun(t, "deploy", nil, "-force", "-cluster", "cluster1", "-tag", "1.2.3")

			assertSingularityRequestTypeService(t, f, customID2)
			assertActiveStatus(t, f, customID2)
			assertNonNilHealthCheckOnLatestDeploy(t, f, customID2)

			// TODO: Implement cleanup of old request.
			//assertRequestDoesNotExist(t, f, customID1)

			assertActiveStatus(t, f, customID1) // This works because we do not yet do cleanup.
		}},
	)
}
