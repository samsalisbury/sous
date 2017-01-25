package graph

import (
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/otpl"
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

func newDetectedOTPLConfig(wd LocalWorkDirShell, otplFlags *config.OTPLFlags) (detectedOTPLDeployManifest, error) {
	if otplFlags.IgnoreOTPLDeploy {
		return detectedOTPLDeployManifest{}, nil
	}
	otplParser := otpl.NewManifestParser()
	otplDeploySpecs := otplParser.ParseManifest(wd.Sh)
	return detectedOTPLDeployManifest{otplDeploySpecs}, nil
}

func newUserSelectedOTPLDeploySpecs(detected detectedOTPLDeployManifest, tmid TargetManifestID, flags *config.OTPLFlags, state *sous.State) (userSelectedOTPLDeployManifest, error) {
	var nowt userSelectedOTPLDeployManifest
	if detected.Manifest == nil {
		return nowt, nil
	}
	mid := sous.ManifestID(tmid)
	// we don't care about these flags when a manifest already exists
	if _, ok := state.Manifests.Get(mid); ok {
		return nowt, nil
	}
	if !flags.UseOTPLDeploy && !flags.IgnoreOTPLDeploy && len(detected.Manifest.Deployments) != 0 {
		return nowt, errors.New("otpl-deploy detected in config/, please specify either -use-otpl-deploy, or -ignore-otpl-deploy to proceed")
	}
	if !flags.UseOTPLDeploy {
		return nowt, nil
	}
	if len(detected.Manifest.Deployments) == 0 {
		return nowt, errors.New("use of otpl configuration was specified, but no valid deployments were found in config/")
	}
	deploySpecs := sous.DeploySpecs{}
	for clusterName, spec := range detected.Manifest.Deployments {
		if _, ok := state.Defs.Clusters[clusterName]; !ok {
			sous.Log.Warn.Printf("otpl-deploy config for cluster %q ignored", clusterName)
			continue
		}
		deploySpecs[clusterName] = spec
	}
	if len(deploySpecs) == 0 {
		return nowt, nil
	}
	// Detach the user selected from the detected manifest, in case something
	// else relies on the detected ones.
	selectedManifest := detected.Manifest.Clone()
	selectedManifest.Deployments = deploySpecs
	return userSelectedOTPLDeployManifest{selectedManifest}, nil
}
