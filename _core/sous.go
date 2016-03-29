package core

import (
	"encoding/json"
	"flag"
	"fmt"

	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/cmd"
	"github.com/opentable/sous/tools/docker"
)

type Sous struct {
	Version, Revision, OS, Arch string
	Packs                       []Pack
	Commands                    map[string]*Command
	cleanupTasks                []func() error
	Flags                       *SousFlags
	Config                      *Config
	State                       *State
	flagSet                     *flag.FlagSet
}

type SousFlags struct {
	ForceRebuild, ForceRebuildAll bool
}

type Command struct {
	Func      func(*Sous, []string)
	HelpFunc  func() string
	ShortDesc string
}

var sous *Sous

func NewSous(version, revision, os, arch string, commands map[string]*Command, packs []Pack, flags *SousFlags, state *State) *Sous {
	var cfg *Config
	if state != nil {
		cfg = &state.Config
	}
	if sous == nil {
		sous = &Sous{
			Version:      version,
			Revision:     revision,
			OS:           os,
			Arch:         arch,
			Packs:        packs,
			Commands:     commands,
			Flags:        flags,
			State:        state,
			Config:       cfg,
			cleanupTasks: []func() error{},
		}
	}
	return sous
}

func (s *Sous) UpdateBaseImage(image string) {
	// First, keep track of which images we are interested in...
	key := "usedBaseImages"
	images := Properties()[key]
	var list []string
	if len(images) != 0 {
		json.Unmarshal([]byte(images), &list)
	} else {
		list = []string{}
	}
	if doesNotAppearInList(image, list) {
		list = append(list, image)
	}
	listJSON, err := json.Marshal(list)
	if err != nil {
		cli.Fatalf("Unable to marshal base image list as JSON: %+v; %s", list, err)
	}
	Set(key, string(listJSON))
	// Now lets grab the actual image
	docker.Pull(image)
}

func (s *Sous) LsImages(c *Context) []*docker.Image {
	labelFilter := fmt.Sprintf("label=%s.build.package.name=%s", s.Config.DockerLabelPrefix, c.CanonicalPackageName())
	results := cmd.Table("docker", "images", "--filter", labelFilter)
	// The first line is just table headers
	if len(results) < 2 {
		return nil
	}
	results = results[1:]
	images := make([]*docker.Image, len(results))
	for i, row := range results {
		images[i] = docker.NewImage(row[0], row[1])
	}
	return images
}

func (s *Sous) LsContainers(c *Context) []docker.Container {
	labelFilter := fmt.Sprintf("label=%s.build.package.name=%s", s.Config.DockerLabelPrefix, c.CanonicalPackageName())
	results := cmd.Table("docker", "ps", "-a", "--filter", labelFilter)
	// The first line is just table headers
	if len(results) < 2 {
		return nil
	}
	results = results[1:]
	containers := make([]docker.Container, len(results))
	for i, row := range results {
		nameIndex := len(row) - 1
		containers[i] = docker.NewContainer(row[0], row[nameIndex])
	}
	return containers
}

func doesNotAppearInList(item string, list []string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}
