package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/SeeSpotRun/coerce"
	"github.com/docopt/docopt-go"
	"github.com/opentable/sous/util/test_with_docker"
)

const (
	docstring = `Set up a docker-based Sous QA environment
Usage: sous_qa_setup [options]

Options:
  -K --shutdown  Rather than set up the QA environment, shut it down
  --timeout=<timeout>  Time allowed before a non-response by services is considered failure [default: 5m]
	--compose-dir=<directory>  The directory containing a 'compose.yaml' file
	--out-path=<path>  The path to write the description of the environment to, or - for stdout [default: -]
`
)

type (
	options struct {
		timeout      string
		timeDuration time.Duration
		composeDir   string
		outPath      string
		shutdown     bool
	}

	// EnvDesc captures the details of the established environment
	EnvDesc struct {
		RegistryName   string
		SingularityURL string
		AgentIP        string
	}
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	opts := parseOpts()
	if opts.shutdown {
		teardownServices(opts)
	}
	desc := setupServices(opts)
	writeOut(opts, desc)
}

func teardownServices(opts *options) {
	testAgent, err := test_with_docker.NewAgentWithTimeout(opts.timeDuration)
	if err != nil {
		log.Fatal(err)
	}
	testAgent.Shutdown(started)
}

func setupServices(opts *options) *EnvDesc {
	testAgent, err := test_with_docker.NewAgentWithTimeout(opts.timeDuration)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		testAgent.Cleanup()
	}()

	desc := EnvDesc{}

	desc.Ip, err = testAgent.IP()
	if err != nil {
		log.Fatal(err)
	}
	if ip == nil {
		log.Fatal(fmt.Errorf("Test agent returned nil IP"))
	}

	desc.RegistryName = fmt.Sprintf("%s:%d", ip, 5000)
	desc.SingularityURL = fmt.Sprintf("http://%s:%d/singularity", ip, 7099)

	err = registryCerts(testAgent, opts.composeDir)
	if err != nil {
		log.Fatal(err)
	}

	started, err := testAgent.ComposeServices(opts.composeDir, map[string]uint{"Singularity": 7099, "Registry": 5000})

	return &desc
}

func writeOut(opts *options, desc *EnvDesc) {
	out := os.Stdout
	if out != "-" {
		out = os.Create(opts.outPath)
	}
	enc := json.NewEncoder(out)
	enc.Encode(desc)
}

func parseOpts() *options {
	parsed, err := docopt.Parse(docstring, nil, true, "", false)
	if err != nil {
		log.Fatal(err)
	}

	opts := options{}
	err = coerce.Struct(&opts, parsed, "-%s", "--%s", "<%s>")
	if err != nil {
		log.Fatal(err)
	}
	opts.timeDuration, err = time.ParseDuration(opts.timeout)
	if err != nil {
		log.Fatal(err)
	}

	return &opts
}
