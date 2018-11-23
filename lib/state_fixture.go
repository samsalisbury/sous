package sous

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"strings"
	"text/template"

	"github.com/samsalisbury/semv"
)

// StateFixtureOpts allows configuring StateFixture calls.
type StateFixtureOpts struct {
	ClusterCount  int
	ManifestCount int

	ClusterSuffix string

	ManifestIDOpts *ManifestIDOpts
}

// DefaultStateFixture provides a dummy state for tests by calling
// StateFixture with the following options:
//
//  StateFixtureOpts{
//	  ClusterCount:  3,
//    ManifestCount: 3,
//  }
func DefaultStateFixture() *State {
	return StateFixture(StateFixtureOpts{
		ClusterCount:  3,
		ManifestCount: 3,
	})
}

// DefaultStateFixtureWithFlavorsOffsets is similar to DefaultStateFixture but
// adds flavorN offsetN to each manifest with repoN.
func DefaultStateFixtureWithFlavorsOffsets() *State {
	// NOTE SS: Many old tests assume this specific formatting.
	return StateFixture(StateFixtureOpts{
		ClusterCount:  3,
		ManifestCount: 3,
		ManifestIDOpts: &ManifestIDOpts{
			RepoFmt:   "github.com/user{{.Idx}}/repo{{.Idx}}",
			DirFmt:    "dir{{.Idx}}",
			FlavorFmt: "flavor{{.Idx}}",
		},
	})
}

// GenerateClusters calls gen count times and returns all the generate clusters.
// gen is passed values from 0 to count-1. You should ensure the Name of each
// cluster is unique, otherwise it will panic. If you leave the name blank,
// clusters will be named "clusterN" where N is in the range 1..count.
func GenerateClusters(count int, gen func(idx int) *Cluster) Clusters {
	cs := Clusters{}
	for i := 0; i < count; i++ {
		c := gen(i)
		if c.Name == "" {
			c.Name = fmt.Sprintf("cluster%d", i+1)
		}
		if _, exists := cs[c.Name]; exists {
			log.Panicf("cluster %q already added", c.Name)
		}
		cs[c.Name] = c
	}
	return cs
}

// DefaultCluster returns the default cluster definition. It has no name.
func DefaultCluster() *Cluster {
	return &Cluster{
		Kind:    "singularity",
		BaseURL: "127.0.0.1:5000",
		Env:     EnvDefaults{},
		Startup: Startup{
			SkipCheck:                 false,
			ConnectDelay:              30,
			Timeout:                   30,
			ConnectInterval:           10,
			CheckReadyProtocol:        "HTTP",
			CheckReadyURIPath:         "/health",
			CheckReadyFailureStatuses: []int{500},
			CheckReadyURITimeout:      2,
			CheckReadyInterval:        2,
			CheckReadyRetries:         256,
		},
		AllowedAdvisories: AllAdvisoryStrings(),
	}

}

// ManifestIDOpts is used to control manifest ID generation.
type ManifestIDOpts struct {
	// RepoFmt, DirFmt, FlavorFmt are the names of the repo, dir and flavor
	// respectively. If they contain the %d formatting directive, it will be
	// replaced with the index of the generated ID. They may only contain it
	// once each.
	RepoFmt, DirFmt, FlavorFmt string
}

func (o *ManifestIDOpts) templates() (repo, dir, flavor *template.Template) {
	repo = template.Must(template.New("repo").Parse(o.RepoFmt))
	dir = template.Must(template.New("dir").Parse(o.DirFmt))
	flavor = template.Must(template.New("flavor").Parse(o.FlavorFmt))
	return
}

// DefaultManifestIDOpts returns the default opts.
// If you supply one of more except funcs they are run on the default
// opts in order before it is returned.
func DefaultManifestIDOpts(except ...func(*ManifestIDOpts)) ManifestIDOpts {
	o := ManifestIDOpts{
		RepoFmt: "github.com/user1/repo{{.Idx}}",
	}
	for _, f := range except {
		f(&o)
	}
	return o
}

// GenerateManifestID generates a ManifestID using the supplied options.
func GenerateManifestID(idx int, o ManifestIDOpts) ManifestID {
	// NOTE SS: This seems overly complex, the justification is to allow
	// existing tests to still work without changing the tests other than
	// having them generate the manifest IDs they expect.
	// It's not possible to use simple fmt.Sprintf calls because many
	// of these tests expect the repo to contain multiple instances of
	// the index.
	// Rather than specialise this to those existing tests, just make it
	// general purpose.
	repoT, dirT, flavorT := o.templates()
	i := struct {
		Idx int
	}{idx + 1}
	repoW, dirW, flavorW := &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}
	if err := repoT.Execute(repoW, i); err != nil {
		panic(err)
	}
	if err := dirT.Execute(dirW, i); err != nil {
		panic(err)
	}
	if err := flavorT.Execute(flavorW, i); err != nil {
		panic(err)
	}
	repo, dir, flavor := repoW.String(), dirW.String(), flavorW.String()

	return ManifestID{
		Source: SourceLocation{
			Repo: repo,
			Dir:  dir,
		},
		Flavor: flavor,
	}
}

// DefaultManifest returns the default manifest.
func DefaultManifest(mid ManifestID) *Manifest {
	return &Manifest{
		Source: mid.Source,
		Flavor: mid.Flavor,
		Owners: nil,
		Kind:   "http-service",
	}
}

// DefaultManifests returns count default manifests.
// If you pass except functions, they are each run in order on every manifest
// created, passing in also the index of that manifest.
func DefaultManifests(count int, except ...func(int, *Manifest)) Manifests {
	return GenerateManifests(count, func(idx int) *Manifest {
		m := DefaultManifest(GenerateManifestID(idx, DefaultManifestIDOpts()))
		for _, f := range except {
			f(idx, m)
		}
		return m
	})
}

// GenerateManifests generates count manifests using the gen function to create
// each one, with the index from 0..count-1 passed in.
func GenerateManifests(count int, gen func(idx int) *Manifest) Manifests {
	m := NewManifests()
	for idx := 0; idx < count; idx++ {
		m.Add(gen(idx))
	}
	return m
}

// StateFixture provides a dummy state for tests.
func StateFixture(o StateFixtureOpts) *State {

	s := NewState()

	c := GenerateClusters(o.ClusterCount, func(idx int) *Cluster {
		c := DefaultCluster()
		c.Name = fmt.Sprintf("cluster%d%s", idx+1, o.ClusterSuffix)
		c.Env["CLUSTER_NAME"] = Var(c.Name)
		return c
	})
	clusterNames := c.Names()
	sort.Strings(clusterNames)

	ms := DefaultManifests(o.ManifestCount, func(idx int, m *Manifest) {
		// TODO SS: set owners nonempty once it is needed for validation.
		m.Owners = []string{}
		if o.ManifestIDOpts != nil {
			id := GenerateManifestID(idx, *o.ManifestIDOpts)
			m.SetID(id)
		}
	})

	// For each cluster add a deployment to each manifest.
	for clusterN := 0; clusterN < o.ClusterCount; clusterN++ {
		clusterName := clusterNames[clusterN]
		for _, mid := range ms.Keys() {
			manifest, ok := ms.Get(mid)
			if !ok {
				panic("Manifests.Keys returned a nonexistent key")
			}

			if manifest.Deployments == nil {
				manifest.Deployments = DeploySpecs{}
			}
			did := DeploymentID{ManifestID: mid, Cluster: clusterName}
			manifest.Deployments[clusterName] = DeploySpec{
				DeployConfig: DeployConfig{
					Resources: map[string]string{
						"cpus":   "0.1",
						"memory": "32",
						"ports":  "1",
					},
					Startup: Startup{
						CheckReadyProtocol: "HTTP",
					},
					Metadata: map[string]string{
						"": "",
					},
					Env: map[string]string{
						"": "",
					},
					NumInstances:         3,
					Volumes:              nil,
					Schedule:             "",
					SingularityRequestID: makeTestFixtureSingularityReqID(did),
				},
				Version: semv.MustParse("1.0.0"),
			}
		}
	}

	s.Defs = Defs{
		DockerRepo: "docker.example.com",
		Clusters:   c,
	}
	s.Manifests = ms

	return s
}

func makeTestFixtureSingularityReqID(did DeploymentID) string {
	repo := strings.Split(did.ManifestID.Source.Repo, "/")
	dir := strings.Replace(did.ManifestID.Source.Dir, "/", "-", -1)
	if dir != "" {
		dir = "-" + dir
	}
	return fmt.Sprintf("new-%s-%s-%s", repo[len(repo)-1]+dir, did.ManifestID.Flavor, did.Cluster)

}
