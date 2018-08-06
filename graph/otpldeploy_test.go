package graph

import (
	"testing"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserSelectedOTPLDeploySpecs(t *testing.T) {
	testcase := func(
		name string,
		DetectedManifest *sous.Manifest,
		Flags config.OTPLFlags,
		TargetMID sous.ManifestID,
		Clusters sous.Clusters,
		ExpectedManifest *sous.Manifest,
	) {
		t.Run(name, func(t *testing.T) {
			state := sous.NewState()
			state.Defs.Clusters = Clusters
			detected := detectedOTPLDeployManifest{}
			if DetectedManifest != nil {
				detected.Manifests = sous.NewManifests(DetectedManifest)
			} else {
				detected.Manifests = sous.NewManifests()
			}
			ls, _ := logging.NewLogSinkSpy()
			graphWrapper := LogSink{ls}
			sm, _ := sous.NewStateManagerSpyFor(state)
			sc := &sous.SourceContext{}
			ds, err := newUserSelectedOTPLDeploySpecs(detected, TargetManifestID(TargetMID), sc, &Flags, &ClientStateManager{StateManager: sm}, graphWrapper)
			assert.NoError(t, err)
			assert.Equal(t, ExpectedManifest, ds.Manifest)
		})
	}

	testcase("no flags no config detected",
		nil,
		config.OTPLFlags{},
		sous.ManifestID{},
		sous.Clusters{},
		nil,
	)

	testcase("detected but ignored so no manifest",
		&sous.Manifest{
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
		config.OTPLFlags{IgnoreOTPLDeploy: true},
		sous.ManifestID{},
		sous.Clusters{
			"some-cluster": nil,
		},
		nil,
	)

	testcase("detected and flags say use",
		&sous.Manifest{
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
		config.OTPLFlags{UseOTPLDeploy: true},
		sous.ManifestID{},
		sous.Clusters{
			"some-cluster": nil,
		},
		&sous.Manifest{
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
			Owners: []string{},
		},
	)

	testcase("detected with flavor and flags say use",
		&sous.Manifest{
			Flavor: "neopolitan",
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
		config.OTPLFlags{UseOTPLDeploy: true, Flavor: "neopolitan"},
		sous.ManifestID{},
		sous.Clusters{
			"some-cluster": nil,
		},
		&sous.Manifest{
			Flavor: "neopolitan",
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
			Owners: []string{},
		},
	)

	// TODO SS: Use nonzero TargetManifestID, right now it fails with that.
	testcase("sourcelocation passthrough with zero TargetManifestID",
		&sous.Manifest{
			Source: sous.SourceLocation{Repo: "repo1", Dir: "dir1"},
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
		config.OTPLFlags{UseOTPLDeploy: true},
		sous.ManifestID{},
		sous.Clusters{
			"some-cluster": {},
		},
		&sous.Manifest{
			Source: sous.SourceLocation{Repo: "repo1", Dir: "dir1"},
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
			Owners: []string{},
		},
	)

}

func TestNewUserSelectedOTPLDeploySpecs_Errors(t *testing.T) {
	testcase := func(
		name string,
		DetectedManifest *sous.Manifest,
		Flags config.OTPLFlags,
		TargetMID sous.ManifestID,
		ExpectedErr string,
	) {
		t.Run(name, func(t *testing.T) {
			state := sous.NewState()
			state.Defs.Clusters = sous.Clusters{}
			detected := detectedOTPLDeployManifest{}
			if DetectedManifest != nil {
				detected.Manifests = sous.NewManifests(DetectedManifest)
			} else {
				detected.Manifests = sous.NewManifests()
			}
			ls, _ := logging.NewLogSinkSpy()
			graphWrapper := LogSink{ls}
			s, _ := sous.NewStateManagerSpyFor(state)
			sc := &sous.SourceContext{}
			ds, err := newUserSelectedOTPLDeploySpecs(detected, TargetManifestID(TargetMID), sc, &Flags, &ClientStateManager{StateManager: s}, graphWrapper)
			assert.Nil(t, ds.Manifest)
			require.Error(t, err)
			assert.Equal(t, err.Error(), ExpectedErr)
		})
	}

	testcase("detected, but no flags set to either use or ignore them",
		&sous.Manifest{
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
		config.OTPLFlags{},
		sous.ManifestID{},
		"otpl-deploy detected in config/, please specify either -use-otpl-deploy, or -ignore-otpl-deploy to proceed",
	)

	testcase("detected with flavor, flags set to use but no flavor specified",
		&sous.Manifest{
			Flavor: "chocolate",
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
		config.OTPLFlags{UseOTPLDeploy: true},
		sous.ManifestID{},
		"flavor \"\" not detected; pick from: [\"chocolate\"]",
	)

	testcase("detected with flavor, flags set to use but unknown flavor specified",
		&sous.Manifest{
			Flavor: "chocolate",
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
		config.OTPLFlags{UseOTPLDeploy: true, Flavor: "strawberry"},
		sous.ManifestID{},
		"flavor \"strawberry\" not detected; pick from: [\"chocolate\"]",
	)

	testcase("not detected but flags expect one",
		nil,
		config.OTPLFlags{UseOTPLDeploy: true},
		sous.ManifestID{},
		"use of otpl configuration was specified, but no valid deployments were found in config/",
	)
}
