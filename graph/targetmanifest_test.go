package graph

import (
	"testing"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUserSelectedOTPLDeploySpecs(t *testing.T) {
	testcase := func(
		name string,
		DetectedManifest *sous.Manifest,
		//XXX,
		Flags config.OTPLFlags,
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

			ds, err := newUserSelectedOTPLDeploySpecs(detected, TargetManifestID{}, &Flags, state)
			assert.NoError(t, err)
			assert.Equal(t, ExpectedManifest, ds.Manifest)
		})
	}

	testcase("no flags no config detected",
		nil,
		config.OTPLFlags{},
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
}

func TestNewUserSelectedOTPLDeploySpecs_Errors(t *testing.T) {
	testcase := func(
		name string,
		DetectedManifest *sous.Manifest,
		Flags config.OTPLFlags,
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

			ds, err := newUserSelectedOTPLDeploySpecs(detected, TargetManifestID{}, &Flags, state)
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
		"flavor \"strawberry\" not detected; pick from: [\"chocolate\"]",
	)

	testcase("not detected but flags expect one",
		nil,
		config.OTPLFlags{UseOTPLDeploy: true},
		"use of otpl configuration was specified, but no valid deployments were found in config/",
	)
}

func TestNewTargetManifest_Existing(t *testing.T) {
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

func TestNewTargetManifest_Existing_withOffset(t *testing.T) {
	detected := userSelectedOTPLDeployManifest{}
	sl := sous.MustParseSourceLocation("github.com/user/project,offset")
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
		Name: "test",
		Kind: "singularity",
		Startup: sous.Startup{
			Timeout:                   180,
			ConnectDelay:              5,
			ConnectInterval:           3,
			CheckReadyProtocol:        "HTTPS",
			CheckReadyURIPath:         "/health",
			CheckReadyFailureStatuses: []int{500, 503},
			CheckReadyInterval:        1,
			CheckReadyRetries:         50,
		},
		BaseURL: "http://singularity.example.com/",
	}
	s.Defs.Clusters = cls
	tm := newTargetManifest(detected, tmid, s)

	s.Manifests.Add(tm.Manifest)

	flaws := s.Validate()

	if len(flaws) > 0 {
		t.Errorf("Invalid new manifest: %#v, flaws were %v", tm.Manifest, flaws)
	}

	var expected = &sous.Manifest{
		Source: sl,
		Kind:   "http-service",
		Flavor: flavor,
		Deployments: sous.DeploySpecs{
			"test": sous.DeploySpec{
				DeployConfig: sous.DeployConfig{
					Resources: sous.Resources{
						"cpus":   "0.1",
						"memory": "100",
						"ports":  "1"},
					NumInstances: 1,
					Startup: sous.Startup{
						CheckReadyProtocol: "HTTP",
						CheckReadyURIPath:  "/health",
					},
					Env: map[string](string){OT_ENV_FLAVOR: flavor},
				},
			},
		},
	}
	assert.Equal(t, expected, tm.Manifest)
}
