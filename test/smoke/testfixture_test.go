//+build smoke

package smoke

import (
	"bufio"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	sous "github.com/opentable/sous/lib"
)

type Fixture struct {
	EnvDesc     desc.EnvDesc
	Cluster     TestCluster
	Client      TestClient
	BaseDir     string
	Singularity *Singularity
}

var originalStdout = os.Stdout
var originalStderr = os.Stderr

func prefixWithTestName(t *testing.T) {
	t.Helper()
	outReader, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Setting up output prefix: %s", err)
	}
	errReader, errWriter, err := os.Pipe()
	if err != nil {
		t.Fatalf("Setting up output prefix: %s", err)
	}
	os.Stdout = outWriter
	os.Stderr = errWriter
	go func() {
		defer func() {
			if err := outReader.Close(); err != nil {
				t.Fatalf("Failed to close outReader: %s", err)
			}
			if err := outWriter.Close(); err != nil {
				t.Fatalf("Failed to close outWriter: %s", err)
			}
		}()
		scanner := bufio.NewScanner(outReader)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				t.Fatalf("Error prefixing stdout: %s", err)
			}
			fmt.Fprintf(originalStdout, "%s::stdout > %s\n", t.Name(), scanner.Text())
		}
	}()
	go func() {
		defer func() {
			if err := errReader.Close(); err != nil {
				t.Fatalf("Failed to close errReader: %s", err)
			}
			if err := errWriter.Close(); err != nil {
				t.Fatalf("Failed to close errWriter: %s", err)
			}
		}()
		scanner := bufio.NewScanner(errReader)
		for scanner.Scan() {
			if err := scanner.Err(); err != nil {
				t.Fatalf("Error prefixing stderr: %s", err)
			}
			fmt.Fprintf(originalStderr, "%s::stderr > %s\n", t.Name(), scanner.Text())
		}
	}()
}

func setupEnv(t *testing.T) Fixture {
	t.Helper()
	if testing.Short() {
		t.Skipf("-short flag present")
	}
	prefixWithTestName(t)
	stopPIDs(t)
	sousBin := getSousBin(t)
	envDesc := getEnvDesc(t)
	baseDir := getDataDir(t)

	singularity := NewSingularity(envDesc.SingularityURL())

	singularity.Reset(t)

	time.Sleep(5 * time.Second)

	state := sous.StateFixture(sous.StateFixtureOpts{
		ClusterCount:  3,
		ManifestCount: 3,
	})

	addURLsToState(state, envDesc)

	c, err := newSmokeTestFixture(t, state, baseDir)
	if err != nil {
		t.Fatalf("setting up test cluster: %s", err)
	}

	if err := c.Configure(envDesc); err != nil {
		t.Fatalf("configuring test cluster: %s", err)
	}

	if err := c.Start(t, sousBin); err != nil {
		t.Fatalf("starting test cluster: %s", err)
	}

	client := makeClient(baseDir, sousBin)
	primaryServer := "http://" + c.Instances[0].Addr
	if err := client.Configure(primaryServer, envDesc.RegistryName()); err != nil {
		t.Fatal(err)
	}

	return Fixture{
		Cluster:     *c,
		Client:      client,
		BaseDir:     baseDir,
		Singularity: singularity,
	}
}

func (f *Fixture) Stop(t *testing.T) {
	t.Helper()
	f.Cluster.Stop(t)
}
