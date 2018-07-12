//+build smoke

package smoke

func fixtureConfigs() []fixtureConfig {
	return []fixtureConfig{
		{dbPrimary: false, projects: projects.SingleDockerfile},
		{dbPrimary: true, projects: projects.SingleDockerfile},
		{dbPrimary: false, projects: projects.SplitBuild},
		{dbPrimary: true, projects: projects.SplitBuild},
	}
}
