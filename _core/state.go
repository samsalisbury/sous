package core

import "fmt"

type State struct {
	Config
	EnvironmentDefs EnvDefs
	Datacentres     Datacentres
	Manifests       Manifests
	Contracts       Contracts
	Buildpacks      Buildpacks
}

type EnvDefs map[string]*EnvDef

type EnvDef map[string]*VarDef

type VarDef struct {
	Type       VarType
	Name, Desc string
	Automatic  bool
}

type VarType string

const (
	URL_VARTYPE    = VarType("url")
	INT_VARTYPE    = VarType("int")
	STRING_VARTYPE = VarType("string")
)

type Datacentres map[string]*Datacentre

type Datacentre struct {
	Name, Desc         string
	SingularityURL     string
	DockerRegistryHost string
	Env                DatacentreEnv
}

type DatacentreEnv map[string]string

type Manifests map[string]Manifest

type Manifest struct {
	App         App
	Deployments Deployments
}

type App struct {
	SourceRepo, Owner, Kind string
}

type Deployments map[string]Deployment

type Deployment struct {
	Instance                  Instance
	SourceTag, SourceRevision string
	Environment               map[string]string
}

type Instance struct {
	Count  int
	CPUs   float32
	Memory string
}

type MemorySize string

func (s *State) ContractsForKind(kind string) (OrderedContracts, error) {
	if list, ok := s.Config.ContractDefs[kind]; ok {
		oc := make(OrderedContracts, len(list))
		for i, contractName := range list {
			if contract, ok := s.Contracts[contractName]; ok {
				oc[i] = contract
			} else {
				return nil, fmt.Errorf("contract %q not defined; it is specified for app kind %s", contractName, kind)
			}
		}
		return oc, nil
	}
	return nil, fmt.Errorf("app kind %s is not defined", kind)
}
