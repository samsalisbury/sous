package core

import (
	"fmt"

	"github.com/samsalisbury/semv"
)

type Config struct {
	DockerRegistry    string
	DockerLabelPrefix string
	GlobalDockerTags  Values
	Packs             *Packs
	Platform          *Platform
	// ContractDefs maps a service kind to an ordered set of contracts
	// to run against apps of that kind.
	ContractDefs map[string]List
}

type Platform struct {
	Services []Service
	EnvDef   []EnvVar
	Envs     []Env
}

type EnvVar struct {
	Name, Type          string
	Required, Protected bool
}

// Service defines a common platform service that most apps will
// rely on. Examples include discovery servers, proxies, config servers, etc.
// These are used in local development, and may be referred to by their name
// in contracts.
type Service struct {
	Name, DockerImage, DockerRunOpts string
}

// Environment defines a named execution environment. This is an open
// ended concept, but a common usage is to have a single environment
// per datacentre, for example.
type Env struct {
	Name string
	Vars map[string]string
}

type Packs struct {
	NodeJS *NodeJSConfig
	Go     *GoConfig
}

type NodeJSConfig struct {
	AvailableVersions    *StackVersions
	DockerTags           map[string]string
	AvailableNPMVersions []string
	DefaultNodeVersion   string
}

type GoConfig struct {
	AvailableVersions *StackVersions
	DefaultGoVersion  string
}

type StackVersions []*StackVersion

type StackVersion struct {
	Name, DefaultImage string
	TargetImages       BaseImageSet
}

type BaseImageSet map[string]string

func (svs StackVersions) VersionList() (semv.VersionList, error) {
	vs := make([]string, len(svs))
	for i, v := range svs {
		vs[i] = v.Name
	}
	return semv.ParseList(vs...)
}

func (sv StackVersion) GetBaseImageTag(target string) string {
	if specificImage, ok := sv.TargetImages[target]; ok {
		return specificImage
	}
	return sv.DefaultImage
}

func (svs StackVersions) GetBestStackVersion(r semv.Range) (*StackVersion, error) {
	vl, err := svs.VersionList()
	if err != nil {
		return nil, err
	}
	v, ok := vl.GreatestSatisfying(r)
	if !ok {
		return nil, fmt.Errorf("version %q not supported", r)
	}
	return svs.GetVersion(v)
}

func (svs StackVersions) GetVersion(version semv.Version) (*StackVersion, error) {
	for _, sv := range svs {
		v, err := semv.Parse(sv.Name)
		if err != nil {
			return nil, err
		}
		if v == version {
			return sv, nil
		}
	}
	return nil, fmt.Errorf("no version matching %s", version)
}
