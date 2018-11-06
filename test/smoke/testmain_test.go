//+build smoke

package smoke

import (
	"flag"
	"os"
	"strings"
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
	wd := "../../integration/test-registry"
	cid, err := doCMDCombinedOut(wd, "docker-compose", "ps", "-q", "registry")
	if err != nil {
		return err
	}
	if strings.TrimSpace(cid) != "" {
		if err := doCMD(wd, "docker-compose", "rm", "-sfv", "registry"); err != nil {
			return err
		}
	}
	vols, err := doCMDCombinedOut(wd, "docker", "volume", "ls")
	if err != nil {
		return err
	}
	if strings.Contains(vols, "test-registry_registrydata") {
		if err := doCMD(wd, "docker", "volume", "rm", "test-registry_registrydata"); err != nil {
			return err
		}
	}
	return doCMD(wd, "docker-compose", "up", "-d", "registry")
}
