package smoke

import sous "github.com/opentable/sous/lib"

type fixtureConfig struct {
	dbPrimary  bool
	startState *sous.State
	projects   projectList
	Desc       string
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

// TODO SS: Remove this from MatrixDef and write a helper func to do the same.
func (m *matrixDef) FixtureConfigs() []fixtureConfig {
	cs := m.combinations()
	fcfgs := make([]fixtureConfig, len(cs))
	for i, c := range m.combinations() {
		m := c.Map()
		fcfgs[i] = fixtureConfig{
			Desc:      c.String(),
			dbPrimary: m["store"].(bool),
			projects:  m["project"].(projectList),
		}
	}
	return fcfgs
}
