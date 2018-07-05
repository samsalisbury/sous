//+build smoke

package main

import (
	"fmt"
	"testing"
)

func TestOTPLInitToDeploy(t *testing.T) {
	pf := pfs.newParallelTestFixture(t)

	fixtureConfigs := []fixtureConfig{
		{dbPrimary: false},
		{dbPrimary: true},
	}

	pf.RunMatrix(fixtureConfigs,
		PTest{Name: "add-artifact", Test: func(t *testing.T, f *TestFixture) {
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
		}},
		PTest{Name: "init-simple", Test: func(t *testing.T, f *TestFixture) {
			client := setupProject(t, f, simpleServer)

		}},
	)
}
