package sous

import "github.com/opentable/sous/util/firsterr"

type (
	// BuildManager collects and orchestrates the various components that are
	// involved with making a build happen
	BuildManager struct {
		BuildConfig *BuildConfig
		Selector
		Labeller
		Registrar
	}
)

// Build implements sous.Builder.Build
func (m *BuildManager) Build() (*BuildResult, error) {
	// TODO if BuildConfig.ForceClone, then clone
	var (
		bp Buildpack
		bc *BuildContext
		br *BuildResult
	)
	err := firsterr.Set(
		func(e *error) { *e = m.BuildConfig.Validate() },
		func(e *error) { bc = m.BuildConfig.NewContext() },
		func(e *error) { *e = m.BuildConfig.GuardStrict(bc) },
		func(e *error) { bp, *e = m.SelectBuildpack(bc) },
		func(e *error) { br, *e = bp.Build(bc) },
		func(e *error) { br.Advisories = bc.Advisories },
		func(e *error) { *e = m.ApplyMetadata(br, bc) },
		func(e *error) { *e = m.BuildConfig.GuardRegister(bc) },
		func(e *error) { *e = m.Register(br, bc) },
	)
	return br, err
}
