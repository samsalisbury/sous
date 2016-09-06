package graph

import (
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/otpl"
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
)

func newDetectedOTPLConfig(wd LocalWorkDirShell, otplFlags *config.OTPLFlags) (DetectedOTPLDeploySpecs, error) {
	if otplFlags.IgnoreOTPLDeploy {
		return DetectedOTPLDeploySpecs{}, nil
	}
	otplParser := otpl.NewDeploySpecParser()
	otplDeploySpecs := otplParser.GetDeploySpecs(wd.Sh)
	return DetectedOTPLDeploySpecs{otplDeploySpecs}, nil
}

func newUserSelectedOTPLDeploySpecs(detected DetectedOTPLDeploySpecs, tmid TargetManifestID, flags *config.OTPLFlags, state *sous.State) (UserSelectedOTPLDeploySpecs, error) {
	var nowt UserSelectedOTPLDeploySpecs
	mid := sous.ManifestID(tmid)
	// we don't care about these flags when a manifest already exists
	if _, ok := state.Manifests.Get(mid); ok {
		return nowt, nil
	}
	if !flags.UseOTPLDeploy && !flags.IgnoreOTPLDeploy && len(detected.DeploySpecs) != 0 {
		return nowt, errors.New("otpl-deploy detected in config/, please specify either -use-otpl-deploy, or -ignore-otpl-deploy to proceed")
	}
	if !flags.UseOTPLDeploy {
		return nowt, nil
	}
	if len(detected.DeploySpecs) == 0 {
		return nowt, errors.New("use of otpl configuration was specified, but no valid deployments were found in config/")
	}
	deploySpecs := sous.DeploySpecs{}
	for clusterName, spec := range detected.DeploySpecs {
		if _, ok := state.Defs.Clusters[clusterName]; !ok {
			sous.Log.Warn.Printf("otpl-deploy config for cluster %q ignored", clusterName)
			continue
		}
		deploySpecs[clusterName] = spec
	}
	return UserSelectedOTPLDeploySpecs{deploySpecs}, nil
}
