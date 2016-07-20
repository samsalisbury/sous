package sous

// BuildManager collects and orchestrates the various components that are
// involved with making a build happen
type BuildManager struct {
	BuildConfig  *BuildConfig
	BuildContext *BuildContext
	BuildPack    Buildpack
	Detect       *DetectResult
	Builder      Builder
	Registrar    Registerer
}

// Build implements sous.Builder.Build
func (m *BuildManager) Build() (*BuildResult, error) {
	br, err := m.BuildPack.Build(m.BuildContext)
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
