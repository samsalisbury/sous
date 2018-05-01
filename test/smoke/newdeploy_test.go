//+build smoke

package smoke

import (
	"sync/atomic"
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

func setMemAndCPUForAll(ds sous.DeploySpecs) sous.DeploySpecs {
	for c := range ds {
		ds[c].Resources["memory"] = "1"
		ds[c].Resources["cpus"] = "0.001"
	}
	return ds
}

func setMinimalMemAndCPUNumInst1(m sous.Manifest) sous.Manifest {
	return transformEachDeployment(m, func(d sous.DeploySpec) sous.DeploySpec {
		d.Resources["memory"] = "1"
		d.Resources["cpus"] = "0.001"
		d.NumInstances = 1
		return d
	})
}

func setMinimalMemAndCPUNumInst0(m sous.Manifest) sous.Manifest {
	return transformEachDeployment(m, func(d sous.DeploySpec) sous.DeploySpec {
		d.Resources["memory"] = "1"
		d.Resources["cpus"] = "0.001"
		d.NumInstances = 0
		return d
	})
}

func transformEachDeployment(m sous.Manifest, f func(sous.DeploySpec) sous.DeploySpec) sous.Manifest {
	for c, d := range m.Deployments {
		m.Deployments[c] = f(d)
	}
	return m
}

// setupProject creates a brand new git repo containing the provided Dockerfile,
// commits that Dockerfile, runs 'sous version' and 'sous config', and returns a
// sous TestClient in the project directory.
func setupProject(t *testing.T, f TestFixture, dockerfile string) *TestClient {
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

	// cd into project dir
	client.Dir = projectDir
	client.ClusterSuffix = f.ClusterSuffix

	// Dump sous version & config.
	t.Logf("Sous version: %s", client.MustRun(t, "version", nil))
	client.MustRun(t, "config", nil)

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

func TestSousDeploy(t *testing.T) {

	resetSingularity(t)

	stopPIDs(t)

	// numFreeAddrs determines the maximum number of parallel sous servers
	// that can be run by the tests. At some point this may need to be increased.
	numFreeAddrs := 128
	freeAddrs := freePortAddrs(t, "127.0.0.1", numFreeAddrs, 6601, 9000)
	var nextAddrIndex int64
	nextAddr := func() string {
		i := atomic.AddInt64(&nextAddrIndex, 1)
		if i == int64(numFreeAddrs) {
			panic("ran out of free ports; increase numFreeAddrs")
		}
		return freeAddrs[i]
	}

	for _, deployCommand := range []string{"newdeploy", "deploy"} {
		t.Run(deployCommand, func(t *testing.T) {
			t.Parallel()

			t.Run("simple", func(t *testing.T) {
				f := newTestFixture(t, nextAddr)
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

			t.Run("zero", func(t *testing.T) {
				f := newTestFixture(t, nextAddr)
				client := setupProject(t, f, simpleServer)
				client.MustRun(t, "init", nil, "-kind", "http-service")
				client.TransformManifest(t, nil, setMinimalMemAndCPUNumInst0)
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

			t.Run("fails", func(t *testing.T) {
				f := newTestFixture(t, nextAddr)
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
				f := newTestFixture(t, nextAddr)
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
				f := newTestFixture(t, nextAddr)
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

				knownToFailHere(t)

				client.MustRun(t, deployCommand, nil, "-cluster", "cluster1", "-tag", "3")
				assertActiveStatus(t, f, did)
			})

			t.Run("scheduled", func(t *testing.T) {
				f := newTestFixture(t, nextAddr)
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
	}
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
