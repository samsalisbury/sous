package docker

import sous "github.com/opentable/sous/lib"

type (
	// A SplitBuildpack implements the pattern of using a build container and producing a separate deploy container
	SplitBuildpack struct {
	}
)

// Detect implements Buildpack on SplitBuildpack
func (sbp *SplitBuildpack) Detect(ctx *sous.BuildContext) (*sous.DetectResult, error) {
	// Look for a Dockerfile
	// Parse for ENV SOUS_MANIFEST
	//   present? -> true
	// Parse for FROM
	// Fetch from-manifest
	// Inspect that for ENV SOUS_MANIFEST
	//   return present?
	return nil, nil
}

// Build implements Buildpack on SplitBuildpack
func (sbp *SplitBuildpack) Build(ctx *sous.BuildContext, drez *sous.DetectResult) (*sous.BuildResult, error) {
	return nil, nil
}
