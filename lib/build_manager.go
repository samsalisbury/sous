package sous

import "github.com/opentable/sous/util/firsterr"

type (
	// BuildManager collects and orchestrates the various components that are
	// involved with making a build happen
	BuildManager struct {
		BuildConfig  *BuildConfig
		BuildContext *BuildContext
		Selector
		Labeller
		Registrar
	}
)

// Build implements sous.Builder.Build
func (m *BuildManager) Build() (br *BuildResult, e error) {
	// TODO if BuildConfig.ForceClone, then clone
	var bp Buildpack

	e = firsterr.Returned(
		func() (e error) { bp, e = m.SelectBuildpack(m.BuildContext); return },
		// XXX not used, and the detect result might come from Select
		// func() (e error) { _, e = bp.Detect(m.BuildContext); return },
		func() (e error) { br, e = bp.Build(m.BuildContext); return },
		func() (e error) { e = m.ApplyMetadata(br); return },
		func() (e error) { e = m.Register(br); return },
	)
	return
}
