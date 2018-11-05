//+build smoke

package smoke

import (
	"flag"
	"os"
	"testing"

	"github.com/opentable/sous/util/firsterr"
	"github.com/opentable/sous/util/testmatrix"
)

func TestMain(m *testing.M) {
	flag.Parse()
	testmatrix.Quiet = quiet()
	sup = testmatrix.Init(matrix, newFixture, func() error {
		return firsterr.Parallel().Set(
			func(e *error) { *e = resetSingularity() },
			func(e *error) { *e = stopPIDs() },
			func(e *error) { *e = nukeDockerRegistry() },
		)
	})
	exitCode := m.Run()
	sup.PrintSummary()
	os.Exit(exitCode)
}

func resetSingularity() error {
	envDesc := getEnvDesc()
	singularity := newSingularity(envDesc.SingularityURL())
	return singularity.Reset()
}

func nukeDockerRegistry() error {
	if err := doCMD("../../integration/test-registry", "docker-compose", "rm", "-sfv", "registry"); err != nil {
		return err
	}
	if err := doCMD(".", "docker", "volume", "rm", "test-registry_registrydata"); err != nil {
		return err
	}
	return doCMD("../../integration/test-registry", "docker-compose", "up", "-d", "registry")
}
