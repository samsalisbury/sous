package graph

import (
	"testing"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
)

type newUserSelectedOTPLDeploySpecTest struct {
	DetectedManifest *sous.Manifest
	//XXX
	TSL              TargetManifestID
	Flags            config.OTPLFlags
	Clusters         sous.Clusters
	ExpectedManifest *sous.Manifest
	ExpectedErr      string
}

var nusodsTests = []newUserSelectedOTPLDeploySpecTest{
	// 0. No flags set and not OTPL config detected, so no errors and no
	//    manifest expected.
	{},

	// 1. Manifest detected, but no flags set to either use or ignore them,
	//    therefore an error and a nil manifest.
	{
		DetectedManifest: &sous.Manifest{
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
		ExpectedErr: "otpl-deploy detected in config/, please specify either -use-otpl-deploy, or -ignore-otpl-deploy to proceed",
	},

	// 2. Manifest detected and flags set to ignore it. Therefore no error and
	//    a nil Manifest.
	{
		DetectedManifest: &sous.Manifest{
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
		Flags: config.OTPLFlags{IgnoreOTPLDeploy: true},
	},

	// 3. Manifest detected, and flags set to use it. Therefore no error
	//    and we expect that manifest to be returned.
	{
		Clusters: sous.Clusters{
			"some-cluster": nil,
		},
		DetectedManifest: &sous.Manifest{
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
		Flags: config.OTPLFlags{UseOTPLDeploy: true},
		ExpectedManifest: &sous.Manifest{
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
	},

	// 4. Manifest detected, but ignored so no error or manifest.
	{
		Clusters: sous.Clusters{
			"some-cluster": nil,
		},
		DetectedManifest: &sous.Manifest{
			Deployments: sous.DeploySpecs{
				"some-cluster": {},
			},
		},
		Flags: config.OTPLFlags{IgnoreOTPLDeploy: true},
	},

	// 5. No manifest detected but flags expect one, so an error about that.
	{
		Flags:       config.OTPLFlags{UseOTPLDeploy: true},
		ExpectedErr: "use of otpl configuration was specified, but no valid deployments were found in config/",
	},
}

func TestNewUserSelectedOTPLDeploySpecs(t *testing.T) {
	for i, test := range nusodsTests {

		//DEBUG
		if i != 3 {
			continue
		}

		state := sous.NewState()

		state.Defs.Clusters = test.Clusters
		ds, err := newUserSelectedOTPLDeploySpecs(
			detectedOTPLDeployManifest{Manifest: test.DetectedManifest},
			test.TSL,
			&test.Flags,
			state,
		)
		if err != nil {
			if test.ExpectedErr == "" {
				t.Error(err)
				continue
			}
			actualErr := err.Error()
			if actualErr != test.ExpectedErr {
				t.Errorf("got error %q; want %q", actualErr, test.ExpectedErr)
			}
			continue
		}
		if err == nil && test.ExpectedErr != "" {
			t.Errorf("got nil; want error %q", test.ExpectedErr)
			continue
		}
		if test.ExpectedManifest == nil && ds.Manifest == nil {
			continue
		}
		if test.ExpectedManifest == nil && ds.Manifest != nil {
			t.Fatalf("%d got a manifest; want nil", i)
		}
		if test.ExpectedManifest != nil && ds.Manifest == nil {
			t.Fatalf("%d got nil; want a manifest", i)
		}
		actualLen := len(ds.Manifest.Deployments)
		expectedLen := len(test.ExpectedManifest.Deployments)
		if actualLen != expectedLen {
			t.Errorf("got %d deploy specs; want %d", actualLen, expectedLen)
		}
	}
}

func TestNewTargetManifest_Existing(t *testing.T) {
	sous.Log.BeChatty()
	defer sous.Log.BeQuiet()
	detected := userSelectedOTPLDeployManifest{}
	sl := sous.MustParseSourceLocation("github.com/user/project")
	flavor := "some-flavor"
	mid := sous.ManifestID{Source: sl, Flavor: flavor}
	tmid := TargetManifestID(mid)
	m := &sous.Manifest{Source: sl, Flavor: flavor, Kind: sous.ManifestKindService}
	s := sous.NewState()
	s.Manifests.Add(m)
	tm := newTargetManifest(detected, tmid, s)
	if tm.Source != sl {
		t.Errorf("unexpected manifest %q", m)
	}
	flaws := tm.Manifest.Validate()
	if len(flaws) > 0 {
		t.Errorf("Invalid existing manifest: %#v, flaws were %v", tm.Manifest, flaws)
	}
}

func TestNewTargetManifest(t *testing.T) {
	detected := userSelectedOTPLDeployManifest{}
	sl := sous.MustParseSourceLocation("github.com/user/project")
	flavor := "some-flavor"
	mid := sous.ManifestID{Source: sl, Flavor: flavor}
	tmid := TargetManifestID(mid)
	s := sous.NewState()
	cls := sous.Clusters{}
	cls["test"] = &sous.Cluster{
		Name:    "test",
		Kind:    "singularity",
		BaseURL: "http://singularity.example.com/",
	}
	s.Defs.Clusters = cls
	tm := newTargetManifest(detected, tmid, s)
	if tm.Manifest == nil {
		return
	}
	flaws := tm.Manifest.Validate()
	if len(flaws) > 0 {
		t.Errorf("Invalid new manifest: %#v, flaws were %v", tm.Manifest, flaws)
	}

}
