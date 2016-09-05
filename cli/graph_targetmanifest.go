package cli

import (
	"github.com/opentable/sous/ext/otpl"
	sous "github.com/opentable/sous/lib"
)

func newDetectedOTPLConfig(wd LocalWorkDirShell, otplFlags *OTPLFlags) (DetectedOTPLDeploySpecs, error) {
	if otplFlags.IgnoreOTPLDeploy {
		return DetectedOTPLDeploySpecs{}, nil
	}
	otplParser := otpl.NewDeploySpecParser()
	otplDeploySpecs := otplParser.GetDeploySpecs(wd.Sh)
	return DetectedOTPLDeploySpecs{otplDeploySpecs}, nil
}

func newUserSelectedOTPLDeploySpecs(detected DetectedOTPLDeploySpecs, tsl TargetManifestID, flags *OTPLFlags, state *sous.State) (UserSelectedOTPLDeploySpecs, error) {
	var nowt UserSelectedOTPLDeploySpecs
	sl := sous.ManifestID(tsl)
	// we don't care about these flags when a manifest already exists
	if _, ok := state.Manifests.Get(sl); ok {
		return nowt, nil
	}
	if !flags.UseOTPLDeploy && !flags.IgnoreOTPLDeploy && len(detected.DeploySpecs) != 0 {
		return nowt, UsageErrorf("otpl-deploy detected in config/, please specify either -use-otpl-deploy, or -ignore-otpl-deploy to proceed")
	}
	if !flags.UseOTPLDeploy {
		return nowt, nil
	}
	if len(detected.DeploySpecs) == 0 {
		return nowt, UsageErrorf("you specified -use-otpl-deploy, but no valid deployments were found in config/")
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

func newTargetManifest(auto UserSelectedOTPLDeploySpecs, tsl TargetManifestID, s *sous.State) TargetManifest {
	mid := sous.ManifestID(tsl)
	//ds := gdm.Filter(func(d *sous.Deployment) bool {
	//	return d.SourceID.Location() == sl
	//})
	m, ok := s.Manifests.Get(mid)
	if ok {
		return TargetManifest{m}
	}
	deploySpecs := auto.DeploySpecs
	if len(deploySpecs) == 0 {
		deploySpecs = defaultDeploySpecs()
	}

	m = &sous.Manifest{
		Deployments: deploySpecs,
	}
	m.SetID(mid)
	return TargetManifest{m}
}
