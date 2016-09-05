package cli

import (
	"testing"

	sous "github.com/opentable/sous/lib"
)

type newUserSelectedOTPLDeploySpecTest struct {
	Detected            sous.DeploySpecs
	TSL                 TargetManifestID
	Flags               OTPLFlags
	Clusters            sous.Clusters
	Manifest            *sous.Manifest
	ExpectedDeploySpecs sous.DeploySpecs
	ExpectedErr         string
}

var nusodsTests = []newUserSelectedOTPLDeploySpecTest{
	{},
	{
		Detected: sous.DeploySpecs{
			"some-cluster": {},
		},
		ExpectedErr: "otpl-deploy detected in config/, please specify either -use-otpl-deploy, or -ignore-otpl-deploy to proceed",
	},
	{
		Detected: sous.DeploySpecs{
			"some-cluster": {},
		},
		Flags: OTPLFlags{IgnoreOTPLDeploy: true},
	},
	{
		Clusters: sous.Clusters{
			"some-cluster": nil,
		},
		Detected: sous.DeploySpecs{
			"some-cluster": {},
		},
		Flags: OTPLFlags{UseOTPLDeploy: true},
		ExpectedDeploySpecs: sous.DeploySpecs{
			"some-cluster": {},
		},
	},
	{
		Clusters: sous.Clusters{
			"some-cluster": nil,
		},
		Detected: sous.DeploySpecs{
			"some-cluster": {},
		},
		Flags: OTPLFlags{IgnoreOTPLDeploy: true},
	},
	{
		Flags:       OTPLFlags{UseOTPLDeploy: true},
		ExpectedErr: "you specified -use-otpl-deploy, but no valid deployments were found in config/",
	},
}

func TestNewUserSelectedOTPLDeploySpecs(t *testing.T) {
	for _, test := range nusodsTests {
		state := sous.NewState()
		if test.Manifest != nil {
			state.Manifests.MustAdd(test.Manifest)
		}
		state.Defs.Clusters = test.Clusters
		ds, err := newUserSelectedOTPLDeploySpecs(
			DetectedOTPLDeploySpecs{test.Detected},
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
		actualLen := len(ds.DeploySpecs)
		expectedLen := len(test.ExpectedDeploySpecs)
		if actualLen != expectedLen {
			t.Errorf("got %d deploy specs; want %d", actualLen, expectedLen)
		}
	}
}

func TestNewTargetManifest(t *testing.T) {
	detected := UserSelectedOTPLDeploySpecs{}
	sl := sous.MustParseSourceLocation("github.com/user/project")
	flavor := "some-flavor"
	mid := sous.ManifestID{Source: sl, Flavor: flavor}
	tsl := TargetManifestID(mid)
	m := &sous.Manifest{Source: sl, Flavor: flavor}
	s := sous.NewState()
	s.Manifests.Add(m)
	tm := newTargetManifest(detected, tsl, s)
	if tm.Source != sl {
		t.Errorf("unexpected manifest %q", m)
	}
}
