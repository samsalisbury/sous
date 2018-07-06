//+build smoke

package main

import (
	"fmt"
	"testing"

	"github.com/opentable/sous/util/filemap"
)

func TestOTPLInitToDeploy(t *testing.T) {
	pf := pfs.newParallelTestFixture(t)

	fixtureConfigs := []fixtureConfig{
		{dbPrimary: false},
		{dbPrimary: true},
	}

	pf.RunMatrix(fixtureConfigs,
		PTest{Name: "add-artifact", Test: func(t *testing.T, f *TestFixture) {
			client := setupProjectSingleDockerfile(t, f, simpleServer)

			reg := f.Client.Config.Docker.RegistryHost
			repo := "github.com/user1/project1"
			tag := "1.2.3"
			dockerTag := f.IsolatedVersionTag(t, tag)
			dockerRepo := fmt.Sprintf("%s/%s", reg, repo)
			dockerRef := fmt.Sprintf("%s:%s", dockerRepo, dockerTag)

			mustDoCMD(t, client.Dir, "docker", "build", "-t", dockerRef, ".")
			mustDoCMD(t, client.Dir, "docker", "push", dockerRef)

			client.MustRun(t, "add artifact", nil, "-docker-image", dockerRepo, "-repo", repo, "-tag", tag)
		}},
		PTest{Name: "init-simple", Test: func(t *testing.T, f *TestFixture) {
			client := setupProject(t, f, simpleServer, filemap.FileMap{
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
					"requestType": "SERVICE",
					"owners": [
					    "test-user1@example.com"
					],
					"slavePlacement": "SEPARATE_BY_REQUEST",
					"instances": 3,
					"rackSensitive": false,
					"loadBalanced": false
				}`,
			})
		}},
		PTest{Name: "init-unknown-fields", Test: func(t *testing.T, f *TestFixture) {
			client := setupProject(t, f, simpleServer, filemap.FileMap{
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

		}},
	)
}
