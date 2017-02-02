package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	docopt "github.com/docopt/docopt-go"
	"github.com/nyarly/coerce"
)

type (
	parameters struct {
		timeout       string
		shutdown      bool
		outPath       string
		composeDir    string
		composeFile   string
		serviceConfig string
	}

	options struct {
		parameters
		out          io.Writer
		timeDuration time.Duration
	}
)

const (
	docstring = `Set up a docker-based Sous QA environment
Usage: sous_qa_setup --compose-dir=<directory> [options]

Options:
   --compose-dir=<directory>    The directory containing a 'docker-compose.yml' file
	 --service-config=<path>      A description of all the service ports you expect your compose to start. [default: services.yml]
	 --compose-file=<path>        Passed to docker-compose [default: docker-compose.yml]
   --timeout=<timeout>          Time allowed before a non-response by services is considered failure [default: 5m]
   --out-path=<path>            The path to write the description of the environment to, or - for stdout [default: -]
   -K --shutdown                Rather than set up the QA environment, shut it down

File paths (the compose, service-config paths) will be considered relative to
the compose-dir. The easiest thing to do is to put your docker-compose.yaml and
services.yaml in the same directory.
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
	if !filepath.IsAbs(opts.parameters.composeFile) {
		opts.composeFile = filepath.Join(opts.composeDir, opts.composeFile)
	}

	info, err := os.Stat(opts.composeFile)
	if err != nil || info.IsDir() {
		return fmt.Errorf("No %q in %q", opts.composeFile, opts.composeDir)
	}

	if rel, err := filepath.Rel(opts.composeDir, opts.composeFile); err == nil {
		opts.composeFile = rel
	}

	if !filepath.IsAbs(opts.serviceConfig) {
		opts.serviceConfig = filepath.Join(opts.composeDir, opts.serviceConfig)
	}

	info, err = os.Stat(opts.serviceConfig)
	if err != nil || info.IsDir() {
		return fmt.Errorf("No %q in %q", opts.serviceConfig, opts.composeDir)
	}

	return nil
}
