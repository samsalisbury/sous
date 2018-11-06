//+build smoke

package smoke

import (
	"flag"
	"log"
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
	log.Printf("Ensuring clean docker registry state.")
	const wd = "../../integration/test-registry"
	cid, err := doCMDCombinedOut(wd, "docker-compose", "ps", "-q", "registry")
	if err != nil {
		return err
	}
	if strings.TrimSpace(cid) != "" {
		log.Printf("Removing registry container (id: %s)", cid)
		if err := doCMD(wd, "docker-compose", "rm", "-sfv", "registry"); err != nil {
			return err
		}
	} else {
		log.Printf("Docker registry is not running.")
	}
	vols, err := doCMDCombinedOut(wd, "docker", "volume", "ls")
	if err != nil {
		return err
	}
	const volumeName = "test-registry_registrydata"
	if strings.Contains(vols, volumeName) {
		log.Printf("Removing registry volume %s", volumeName)
		if err := doCMD(wd, "docker", "volume", "rm", volumeName); err != nil {
			return err
		}
	} else {
		log.Printf("No registry volume named %q to remove.", volumeName)
	}
	log.Printf("Starting a fresh registry container.")
	return doCMD(wd, "docker-compose", "up", "-d", "registry")
}
