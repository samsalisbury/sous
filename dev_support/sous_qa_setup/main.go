// This tool sets up a Sous-specific QA environment. I think, actually, it
// could be made more generic by pushing configuration files into the
// test_registry directory, which might be worthwhile in idle moments
package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"

	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	"github.com/opentable/sous/util/test_with_docker"
	"github.com/samsalisbury/yaml"
)

func main() {
	log.SetFlags(0)
	opts, err := parseOpts()
	if err != nil {
		log.Fatal(err)
	}
	if opts.debug {
		log.SetFlags(log.Lshortfile | log.Ltime)
		log.Print("Debug mode enabled")
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
	if desc.AgentIP, err = testAgent.IP(); err != nil {
		log.Fatal(err)
	}
	if desc.AgentIP == nil {
		log.Fatal(fmt.Errorf("Test agent returned nil IP"))
	}

	desc.RegistryName = fmt.Sprintf("%s:%d", desc.AgentIP, 5000)
	desc.SingularityURL = fmt.Sprintf("http://%s:%d/singularity", desc.AgentIP, 7099)
	desc.GitOrigin = fmt.Sprintf("%s:%d", desc.AgentIP, 2222)

	if err := registryCerts(testAgent, opts.composeDir, desc); err != nil {
		log.Fatal(err)
	}

	var serviceMap map[string]uint
	bytes, err := ioutil.ReadFile(opts.serviceConfig)
	if err != nil {
		log.Fatal(err)
	}
	yaml.Unmarshal(bytes, &serviceMap)
	if _, err := testAgent.ComposeServices(opts.composeDir, serviceMap); err != nil {
		log.Fatal(err)
	}

	return &desc
}

func writeOut(opts *options, desc *desc.EnvDesc) {
	enc := json.NewEncoder(opts.out)
	enc.Encode(desc)
}
