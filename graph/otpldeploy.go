package graph

import (
	"fmt"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/otpl"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/pkg/errors"
)

func newDetectedOTPLConfig(ls LogSink, wd LocalWorkDirShell, otplFlags *config.OTPLFlags) (detectedOTPLDeployManifest, error) {
	if otplFlags.IgnoreOTPLDeploy {
		return detectedOTPLDeployManifest{sous.NewManifests()}, nil
	}
	otplParser := otpl.NewManifestParser(ls)
	otplDeploySpecs := otplParser.ParseManifests(wd.Sh)
	return detectedOTPLDeployManifest{otplDeploySpecs}, nil
}

func newUserSelectedOTPLDeploySpecs(
	detected detectedOTPLDeployManifest,
	tmid TargetManifestID,
	flags *config.OTPLFlags,
	state *sous.State,
) (userSelectedOTPLDeployManifest, error) {
	var nowt userSelectedOTPLDeployManifest

	if detected.Manifests.Len() == 0 {
		if flags.UseOTPLDeploy {
			return nowt, errors.New("use of otpl configuration was specified, but no valid deployments were found in config/")
		}
		return nowt, nil
	}
	mid := sous.ManifestID(tmid)
	// we don't care about these flags when a manifest already exists
	if _, ok := state.Manifests.Get(mid); ok {
		return nowt, nil
	}

	var detectedManifest *sous.Manifest

	if flavoredManifest, ok := detected.Manifests.Single(func(m *sous.Manifest) bool {
		return m.Flavor == flags.Flavor
	}); ok {
		detectedManifest = flavoredManifest
	} else {
		flavors := detected.Manifests.Flavors()
		if flags.Flavor == "" {
			defer messages.ReportLogFieldsMessageToConsole("use the -flavor flat to pick a flavor", logging.WarningLevel, logging.Log)
		}
		return nowt, fmt.Errorf("flavor %q not detected; pick from: %q", flags.Flavor, flavors)
	}

	if !flags.UseOTPLDeploy && !flags.IgnoreOTPLDeploy && len(detectedManifest.Deployments) != 0 {
		return nowt, errors.New("otpl-deploy detected in config/, please specify either -use-otpl-deploy, or -ignore-otpl-deploy to proceed")
	}
	if !flags.UseOTPLDeploy {
		return nowt, nil
	}
	if len(detectedManifest.Deployments) == 0 {
		return nowt, errors.New("use of otpl configuration was specified, but no valid deployments were found in config/")
	}
	deploySpecs := sous.DeploySpecs{}
	for clusterName, spec := range detectedManifest.Deployments {
		if _, ok := state.Defs.Clusters[clusterName]; !ok {
			messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("otpl-deploy config for cluster %q ignored", clusterName), logging.WarningLevel, logging.Log)
			continue
		}
		deploySpecs[clusterName] = spec
	}
	if len(deploySpecs) == 0 {
		return nowt, nil
	}
	// Detach the user selected from the detected manifest, in case something
	// else relies on the detected ones.
	selectedManifest := detectedManifest.Clone()
	selectedManifest.Deployments = deploySpecs
	return userSelectedOTPLDeployManifest{selectedManifest}, nil
}
