//+build smoke

package smoke

import (
	"fmt"
	"testing"

	"github.com/opentable/sous/util/filemap"
)

func dockerBuildAddArtifact(t *testing.T, f *TestFixture, client *TestClient, flags *sousFlags) {
	t.Helper()
	tag := flags.tag
	if tag == "" {
		t.Fatalf("you must add a non-empty tag flag")
	}
	reg := client.Config.Docker.RegistryHost
	repo := "github.com/user1/project1"
	dockerTag := f.IsolatedVersionTag(t, tag)
	dockerRepo := fmt.Sprintf("%s/%s", reg, repo)
	dockerRef := fmt.Sprintf("%s:%s", dockerRepo, dockerTag)

	mustDoCMD(t, client.Dir, "docker", "build", "-t", dockerRef, ".")
	mustDoCMD(t, client.Dir, "docker", "push", dockerRef)

	client.MustRun(t, "artifact add", nil, "-docker-image", dockerRepo, "-repo", repo, "-tag", tag)
}

func TestOTPLInitToDeploy(t *testing.T) {

	t.Skipf("WIP Test")

	pf := pfs.newParallelTestFixture(t)

	fixtureConfigs := []fixtureConfig{
		{dbPrimary: false},
		{dbPrimary: true},
	}

	pf.RunMatrix(fixtureConfigs,

		PTest{Name: "artifact-add", Test: func(t *testing.T, f *TestFixture) {
			client := setupProjectSingleDockerfile(t, f, simpleServer)

			flags := &sousFlags{tag: "1.2.3"}

			dockerBuildAddArtifact(t, f, client, flags)
			// TODO: Assertion that artifact was registered.
		}},

		PTest{Name: "build-init-deploy", Test: func(t *testing.T, f *TestFixture) {
			client := setupProject(t, f, filemap.FileMap{
				"Dockerfile": simpleServer,
				"config/cluster1/singularity.json": `
				{
					"requestId": "request1",
					"resources": {
						"cpus": 0.01,
						"memoryMb": 1,
						"numPorts": 3
					}
				}`,
				"config/cluster1/singularity-request.json": `
				{
					"id": "request1",
					"requestType": "SERVICE",
					"owners": [
					    "test-user1@example.com"
					],
					"instances": 3
				}`,
			})

			flags := &sousFlags{
				kind:    "http-service",
				repo:    "github.com/build-init-deploy-user/project1",
				tag:     "1.2.3",
				cluster: "cluster1",
			}

			dockerBuildAddArtifact(t, f, client, flags)

			client.MustRun(t, "init", flags.SousInitFlags(), "-use-otpl-deploy")

			client.MustRun(t, "deploy", flags.SousDeployFlags())
		}},

		PTest{Name: "fail-unknown-fields", Test: func(t *testing.T, f *TestFixture) {
			client := setupProject(t, f, filemap.FileMap{
				"Dockerfile": simpleServer,
				"config/cluster1/singularity.json": `
				{
					"requestId": "request1",
					"resources": {
						"cpus": 0.01,
						"memoryMb": 1,
						"numPorts": 3
					}
				}`,
				"config/cluster1/singularity-request.json": `
				{
					id: "request1",
					"requestType": "WORKER",
					"owners": [
					    "test-user1@example.com"
					],
					"slavePlacement": "SEPARATE_BY_REQUEST",
					"instances": 3,
					"rackSensitive": false,
					"loadBalanced": false
				}`,
			})
			client.MustRun(t, "version", nil)
		}},
	)
}
