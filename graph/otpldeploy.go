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

func newDetectedOTPLConfig(ls LogSink, wd LocalWorkDirShell, otplFlags *config.OTPLFlags, sc *sous.SourceContext) detectedOTPLDeployManifest {
	if otplFlags.IgnoreOTPLDeploy {
		return detectedOTPLDeployManifest{sous.NewManifests()}
	}

	otplParser := otpl.NewManifestParser(ls)
	otplDeploySpecs, err := otplParser.ParseManifests(wd.Sh)
	if err != nil {
		// This is OK, we are detecting these speculatively.
		return detectedOTPLDeployManifest{sous.NewManifests()}
	}

	ms := sous.NewManifests()

	for _, man := range otplDeploySpecs.Snapshot() {
		man.Source.Dir = sc.OffsetDir
		ms.Add(man)
	}

	return detectedOTPLDeployManifest{ms}
}

func newUserSelectedOTPLDeploySpecs(
	detected detectedOTPLDeployManifest,
	tmid TargetManifestID,
	sc *sous.SourceContext,
	flags *config.OTPLFlags,
	sm *ClientStateManager,
	ls LogSink,
) (userSelectedOTPLDeployManifest, error) {
	var nowt userSelectedOTPLDeployManifest

	state, err := sm.ReadState()
	if err != nil {
		return userSelectedOTPLDeployManifest{}, err
	}

	if detected.Manifests.Len() == 0 {
		if flags.UseOTPLDeploy {
			return nowt, errors.New("use of otpl configuration was specified, but no valid deployments were found in config/")
		}
		return nowt, nil
	}

	if tmid.Source.Dir != sc.OffsetDir {
		// TODO SS: Maybe support specifying other offsets eventually, for now
		// it's hard to know the user's intention (via flags).
		return nowt, fmt.Errorf("the offset of the current directory is %q; but you specified %q", sc.OffsetDir, tmid.Source.Dir)
	}

	mid := sous.ManifestID(tmid)
	// we don't care about these flags when a manifest already exists
	if _, ok := state.Manifests.Get(mid); ok {
		return nowt, fmt.Errorf("manifest %s already exists", mid)
	}

	selected, err := getSelectedManifest(detected.Manifests, flags, ls)
	if err != nil {
		return nowt, err
	}

	if !flags.UseOTPLDeploy && !flags.IgnoreOTPLDeploy && len(selected.Deployments) != 0 {
		return nowt, errors.New("otpl-deploy detected in config/, please specify either -use-otpl-deploy, or -ignore-otpl-deploy to proceed")
	}
	if !flags.UseOTPLDeploy {
		return nowt, nil
	}
	if len(selected.Deployments) == 0 {
		return nowt, errors.New("use of otpl configuration was specified, but no valid deployments were found in config/")
	}
	deploySpecs := sous.DeploySpecs{}
	for clusterName, spec := range selected.Deployments {
		if _, ok := state.Defs.Clusters[clusterName]; !ok {
			// TODO SS: Error if clusters not recognised.
			messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("otpl-deploy config for cluster %q ignored", clusterName), logging.WarningLevel, ls)
			continue
		}
		deploySpecs[clusterName] = spec
	}
	if len(deploySpecs) == 0 {
		return nowt, nil
	}
	// Detach the user selected from the detected manifest, in case something
	// else relies on the detected ones.
	selectedManifest := selected.Clone()
	selectedManifest.Deployments = deploySpecs
	return userSelectedOTPLDeployManifest{selectedManifest}, nil
}

func getSelectedManifest(detected sous.Manifests, flags *config.OTPLFlags, ls logging.LogSink) (*sous.Manifest, error) {
	if flavoredManifest, ok := detected.Single(func(m *sous.Manifest) bool { return m.Flavor == flags.Flavor }); ok {
		return flavoredManifest, nil
	}
	flavors := detected.Flavors()
	if flags.Flavor == "" {
		defer messages.ReportLogFieldsMessageToConsole("use the -flavor flag to pick a flavor", logging.WarningLevel, ls)
	}
	return nil, fmt.Errorf("flavor %q not detected; pick from: %q", flags.Flavor, flavors)
}
