package smoke

import sous "github.com/opentable/sous/lib"

type fixtureConfig struct {
	dbPrimary  bool
	startState *sous.State
	projects   projectList
}

// matrix returns the defined sous smoke test matrix.
func matrix() matrixDef {
	m := newMatrix()
	m.AddDimension("store", "GDM storage to use", map[string]interface{}{
		"db":  true,
		"git": false,
	})
	m.AddDimension("project", "type of project to build", map[string]interface{}{
		"simple": projects.SingleDockerfile,
		"split":  projects.SplitBuild,
	})
	return m
}

func makeFixtureConfig(c combination) fixtureConfig {
	m := c.Map()
	return fixtureConfig{
		dbPrimary: m["store"].(bool),
		projects:  m["project"].(projectList),
	}
}
