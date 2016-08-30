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
func (m *BuildManager) Build() (br *BuildResult, e error) {
	// TODO if BuildConfig.ForceClone, then clone
	var bp Buildpack
	var bc *BuildContext

	e = firsterr.Returned(
		func() (e error) { return m.BuildConfig.Validate() },
		func() (e error) { bc = m.BuildConfig.NewContext(); return nil },
		func() (e error) { e = m.BuildConfig.GuardStrict(bc); return },
		func() (e error) { bp, e = m.SelectBuildpack(bc); return },
		func() (e error) { br, e = bp.Build(bc); return },
		func() (e error) { br.Advisories = bc.Advisories; return nil },
		func() (e error) { e = m.ApplyMetadata(br, bc); return },
		func() (e error) { e = m.BuildConfig.GuardRegister(bc); return },
		func() (e error) { e = m.Register(br, bc); return },
	)
	return
}
