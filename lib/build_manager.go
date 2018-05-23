package sous

import (
	"path/filepath"
	"strings"

	"github.com/opentable/sous/util/firsterr"
	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
)

type (
	// BuildManager collects and orchestrates the various components that are
	// involved with making a build happen
	BuildManager struct {
		BuildConfig *BuildConfig
		Selector
		Labeller
		Registrar
		LogSink logging.LogSink
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
	err := firsterr.Panic(
		func(e *error) { *e = m.BuildConfig.Validate() },
		func(e *error) { bc = m.BuildConfig.NewContext() },
		func(e *error) { *e = m.BuildConfig.GuardStrict(bc) },
		func(e *error) { bp, *e = m.SelectBuildpack(bc) },
		func(e *error) { br, *e = bp.Build(bc) },
		func(e *error) { br.Contextualize(bc) },
		func(e *error) { *e = m.ApplyMetadata(br) },
		func(e *error) { *e = m.RegisterAndWarnAdvisories(br) },
	)
	return br, errors.Wrap(err, "unable to build")
}

// RegisterAndWarnAdvisories registers the image if there are no blocking
// advisories; warns about the advisories and does not register otherwise.
func (m *BuildManager) RegisterAndWarnAdvisories(br *BuildResult) error {
	if err := m.BuildConfig.GuardRegister(br); err != nil {
		logging.ReportError(m.LogSink, err)
	}
	return m.Register(br)
}

// OffsetFromWorkdir sets the offset for the BuildManager to be the indicated directory.
// It's a convenience for command line users who can `sous build <dir>` (and therefore get tab-completion etc)
func (m *BuildManager) OffsetFromWorkdir(offset string) error {
	cfg := m.BuildConfig
	if cfg.Offset != "" { // because --offset
		return errors.New("Cannot use both --offset and path argument")
	}
	sc := cfg.Context.Source
	workdir := sc.OffsetDir

	workAbs := filepath.Join(sc.RootDir, workdir)

	if !filepath.IsAbs(offset) {
		offset = filepath.Join(workAbs, offset)
	}

	offset, err := filepath.Rel(sc.RootDir, offset)

	if err != nil {
		return errors.Wrap(err, "offset")
	}
	if strings.HasPrefix(offset, "..") {
		return errors.Errorf("Offset %q outside of project root %q", offset, sc.RootDir)
	}
	if offset == "." {
		offset = ""
	}

	cfg.Offset = offset
	return nil
}
