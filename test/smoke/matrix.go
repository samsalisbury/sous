package smoke

import (
	"github.com/opentable/sous/util/testmatrix"
)

// matrix returns the defined sous smoke test matrix.
func matrix() testmatrix.Matrix {
	return testmatrix.New(
		testmatrix.Dimension{
			Name: "store",
			Desc: "GDM storage to use",
			Values: map[string]interface{}{
				"db":  true,
				"git": false,
			},
		},
		testmatrix.Dimension{
			Name: "project",
			Desc: "type of project to build",
			Values: map[string]interface{}{
				"simple": projects.SingleDockerfile,
				"split":  projects.SplitBuild,
			},
		},
	)
}

// scenario is an unwrapped testmatrix.Scenario formed from the matrix
// definition returned by matrix().
type scenario struct {
	dbPrimary bool
	projects  projectList
}

// unwrapScenario transforms a generic testmatrix.Scenario into a strongly typed
// scenario for use in this test package.
func unwrapScenario(c testmatrix.Scenario) scenario {
	m := c.Map()
	return scenario{
		dbPrimary: m["store"].(bool),
		projects:  m["project"].(projectList),
	}
}
