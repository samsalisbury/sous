//+build smoke

package smoke

import "testing"

// These funcs I'm calling macros fast-forward tests to interesting states.

// initProject inits a project and transforms it manifest.
func initProject(t *testing.T, client *sousProject, flags *sousFlags, transforms ...ManifestTransform) {
	client.MustRun(t, "init", flags.SousInitFlags())
	client.TransformManifest(t, flags.ManifestIDFlags(), transforms...)
}

// initBuild is a macro to fast-forward to tests to a point where we have
// initialised the project, transformed the manifest, and performed a build.
func initBuild(t *testing.T, client *sousProject, flags *sousFlags, transforms ...ManifestTransform) {
	initProject(t, client, flags, transforms...)
	client.MustRun(t, "build", flags.SourceIDFlags())
}

// initBuildDeploy is a macro for fast-forwarding tests to a point where we have
// initialised the project as a kind project, built it with -tag tag and
// deployed that tag successfully to cluster.
func initBuildDeploy(t *testing.T, client *sousProject, flags *sousFlags, transforms ...ManifestTransform) {
	initBuild(t, client, flags, transforms...)
	client.MustRun(t, "deploy", flags.SousDeployFlags())
}

// assumeSuccessfullyDeployed runs initBuildDeploy and asserts deployment was
// successful using the provided singularity reqID.
func assumeSuccessfullyDeployed(t *testing.T, f *fixture, p *sousProject, flags *sousFlags, reqID string) {
	initBuildDeploy(t, p, flags, setMinimalMemAndCPUNumInst1)
	assertActiveStatus(t, f, reqID)
	assertSingularityRequestTypeService(t, f, reqID)
	assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
}
