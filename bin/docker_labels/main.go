package main

import (
	"fmt"
	"log"

	"github.com/docopt/docopt-go"
	"github.com/opentable/sous/util/docker_registry"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	parsed, err := docopt.Parse(cleanWS(`
	Usage:
	  docker_labels [options] <image-name>

	Options:
	  --insecure  makes the connection to e.g. a self-signed registry
	`), nil, true, "", false)

	if err != nil {
		log.Fatal(err)
	}

	imageName := parsed["<image-name>"].(string)
	client := docker_registry.NewClient()
	if _, ok := parsed["--insecure"]; ok {
		client.BecomeFoolishlyTrusting()
	}

	labels, err := client.LabelsForImageName(imageName)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d labels:\n", len(labels))
	for key, value := range labels {
		fmt.Printf("%s: %s\n", key, value)
	}
}
