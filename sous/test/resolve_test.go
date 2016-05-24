package test

import (
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/test_with_docker"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestResolve(t *testing.T) {
	assert := assert.New(t)

	clusterDefs := sous.Defs{
		Clusters: sous.Clusters{
			"it": sous.Cluster{
				BaseURL: registryName,
			},
		},
	}

	stateOneTwo := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"one": manifest("https://github.com/opentable/one", "1.1.1"),
			"two": manifest("https://github.com/opentable/two", "1.1.1"),
		},
	}
	stateTwoThree := sous.State{
		Defs: clusterDefs,
		Manifests: sous.Manifests{
			"two":   manifest("https://github.com/opentable/two", "1.1.1"),
			"three": manifest("https://github.com/opentable/three", "1.1.1"),
		},
	}

	Resolve(stateOneTwo)
	// one and two are running
	Resolve(stateTwoThree)
	// two and three are running, not one

}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(wrapCompose(m))
}

func manifest(sourceURL, version string) sous.Manifest {
	return sous.Manifest{
		Source: sous.SourceLocation{
			RepoURL:    sous.RepoURL(sourceURL),
			RepoOffset: sous.RepoOffset(""),
		},
		Owners: []string{`xyz`},
		Kind:   sous.ManifestKindService,
		Deployments: sous.DeploySpecs{
			"it": sous.PartialDeploySpec{
				DeployConfig: sous.DeployConfig{
					Resources:    sous.Resouces{}, //map[string]string
					Args:         []string{},
					Env:          sous.Env{}, //map[s]s
					NumInstances: 1,
				},
				Version:     semv.MustParse(version),
				clusterName: "it",
			},
		},
	}
}

func wrapCompose(m *testing.M) (resultCode int) {
	log.SetFlags(log.Flags() | log.Lshortfile)

	if testing.Short() {
		return 0
	}

	defer func() {
		log.Println("Cleaning up...")
		if err := recover(); err != nil {
			log.Print("Panic: ", err)
			resultCode = 1
		}
	}()

	testAgent, err := test_with_docker.NewAgentWithTimeout(5 * time.Minute)
	if err != nil {
		panic(err)
	}

	ip, err := testAgent.IP()
	if err != nil {
		panic(err)
	}

	composeDir := "test-registry"

	registryName = fmt.Sprintf("%s:%d", ip, 5000)

	err = registryCerts(testAgent, composeDir)

	started, err := testAgent.ComposeServices(composeDir, map[string]uint{"Singularity": 7099, "Registry": 5000})
	defer testAgent.Shutdown(started)

	log.Print("   *** Beginning tests... ***\n\n")
	resultCode = m.Run()
	return
}
