// This tool sets up a Sous-specific QA environment. I think, actually, it
// could be made more generic by pushing configuration files into the
// test_registry directory, which might be worthwhile in idle moments
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	"github.com/SeeSpotRun/coerce"
	"github.com/docopt/docopt-go"
	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	"github.com/opentable/sous/util/test_with_docker"
)

type (
	parameters struct {
		timeout    string
		shutdown   bool
		outPath    string
		composeDir string
	}

	options struct {
		parameters
		out          io.Writer
		timeDuration time.Duration
	}
)

const (
	docstring = `Set up a docker-based Sous QA environment
Usage: sous_qa_setup [options]

Options:
   --compose-dir=<directory>    The directory containing a 'docker-compose.yaml' file
   --timeout=<timeout>          Time allowed before a non-response by services is considered failure [default: 5m]
   --out-path=<path>            The path to write the description of the environment to, or - for stdout [default: -]
   -K --shutdown                Rather than set up the QA environment, shut it down
`
)

func parseOpts() (*options, error) {
	parsed, err := docopt.Parse(docstring, nil, true, "", false)
	if err != nil {
		return nil, err
	}

	parms := parameters{}
	err = coerce.Struct(&parms, parsed, "-%s", "--%s", "<%s>")
	if err != nil {
		return nil, err
	}

	if parms.composeDir == "" {
		return nil, fmt.Errorf("--compose-dir is required")
	}

	opts := options{parameters: parms}
	opts.timeDuration, err = time.ParseDuration(opts.timeout)
	if err != nil {
		return nil, err
	}

	opts.out = os.Stdout
	if opts.outPath != "-" {
		opts.out, err = os.Create(opts.outPath)
		if err != nil {
			return nil, err
		}
	}

	err = checkCompDir(&opts)
	if err != nil {
		return nil, err
	}

	return &opts, nil
}

func checkCompDir(opts *options) error {
	for _, composeName := range []string{"docker-compose.yaml", "docker-compose.yml"} {
		info, err := os.Stat(path.Join(opts.composeDir, composeName))
		if err == nil && !info.IsDir() {
			return nil
		}
	}

	return fmt.Errorf("No docker-compose.yaml in %q", opts.composeDir)
}

func main() {
	log.SetFlags(0)
	opts, err := parseOpts()
	if err != nil {
		log.Fatal(err)
	}
	testAgent := buildAgent(opts)
	defer func() { testAgent.Cleanup() }()

	if opts.shutdown {
		teardownServices(testAgent, opts)
		return
	}
	desc := setupServices(testAgent, opts)
	writeOut(opts, desc)
}

func buildAgent(opts *options) test_with_docker.Agent {
	testAgent, err := test_with_docker.NewAgentWithTimeout(opts.timeDuration)
	if err != nil {
		log.Fatal(err)
	}
	return testAgent
}

func teardownServices(testAgent test_with_docker.Agent, opts *options) {
	testAgent.ShutdownNow()
}

func setupServices(testAgent test_with_docker.Agent, opts *options) *desc.EnvDesc {
	desc := desc.EnvDesc{}
	var err error

	desc.AgentIP, err = testAgent.IP()
	if err != nil {
		log.Fatal(err)
	}
	if desc.AgentIP == nil {
		log.Fatal(fmt.Errorf("Test agent returned nil IP"))
	}

	desc.RegistryName = fmt.Sprintf("%s:%d", desc.AgentIP, 5000)
	desc.SingularityURL = fmt.Sprintf("http://%s:%d/singularity", desc.AgentIP, 7099)
	desc.GitOrigin = fmt.Sprintf("%s:%d", desc.AgentIP, 2222)

	err = registryCerts(testAgent, opts.composeDir, desc)
	if err != nil {
		log.Fatal(err)
	}

	_, err = testAgent.ComposeServices(opts.composeDir, map[string]uint{"Singularity": 7099, "Registry": 5000, "Git": 2222})
	if err != nil {
		log.Fatal(err)
	}

	return &desc
}

func writeOut(opts *options, desc *desc.EnvDesc) {
	enc := json.NewEncoder(opts.out)
	enc.Encode(desc)
}
