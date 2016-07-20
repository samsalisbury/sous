package sous

import "github.com/opentable/sous/ext/docker"

// BuildManager collects and orchestrates the various components that are
// involved with making a build happen
type BuildManager struct {
	BuildConfig  *BuildConfig
	BuildContext *BuildContext
	Builder      Builder
	Registrar    Registerer
}

// Build implements sous.Builder.Build
func (m *BuildManager) Build() (*BuildResult, error) {
	// TODO if BuildConfig.ForceClone, then clone

	bp := docker.NewDockerfileBuildpack()
	_, err := bp.Detect(m.BuildContext)
	if err != nil {
		return nil, err
	}

	br, err := bp.Build(m.BuildContext)
	if err != nil {
		return nil, err
	}

	err = m.Builder.ApplyMetadata(br)
	if err != nil {
		return nil, err
	}

	err = m.Registrar.Register(br)
	if err != nil {
		return nil, err
	}

	return br, nil
}
