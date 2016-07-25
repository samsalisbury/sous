package sous

import "github.com/opentable/sous/util/firsterr"

type (
	// A Selector selects the buildpack for a given build context
	Selector interface {
		SelectBuildpack(*BuildContext) Buildpack
	}

	// BuildManager collects and orchestrates the various components that are
	// involved with making a build happen
	BuildManager struct {
		BuildConfig  *BuildConfig
		BuildContext *BuildContext
		Selector
		Builder
		Registrar
	}
)

// Build implements sous.Builder.Build
func (m *BuildManager) Build() (br *BuildResult, e error) {
	// TODO if BuildConfig.ForceClone, then clone

	bp := m.SelectBuildpack(m.BuildContext)
	e = firsterr.Returned(
		func() (e error) { _, e = bp.Detect(m.BuildContext); return },
		func() (e error) { br, e = bp.Build(m.BuildContext); return },
		func() (e error) { e = m.ApplyMetadata(br); return },
		func() (e error) { e = m.Register(br); return },
	)
	return
}
