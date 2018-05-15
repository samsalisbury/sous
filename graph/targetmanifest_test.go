package graph

import (
	"fmt"
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
			ls, _ := logging.NewLogSinkSpy()
			graphWrapper := LogSink{ls}
			ds, err := newUserSelectedOTPLDeploySpecs(detected, TargetManifestID{}, &Flags, state, graphWrapper)
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
			ls, _ := logging.NewLogSinkSpy()
			graphWrapper := LogSink{ls}
			ds, err := newUserSelectedOTPLDeploySpecs(detected, TargetManifestID{}, &Flags, state, graphWrapper)
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
	sl := sous.SourceLocation{
		Repo: "github.com/user/project",
		Dir:  "server",
	}
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

func TestNewRefinedResolveFilter(t *testing.T) {
	type In struct {
		Filter     *sous.ResolveFilter
		Discovered *SourceContextDiscovery
	}

	rrfTests := []struct {
		Desc        string
		In          In
		ExpectPanic bool
		ExpectErr   string
		Expect      func(*RefinedResolveFilter) error
	}{
		{
			Desc:        "nil inputs panics",
			ExpectPanic: true,
		},
		{
			Desc: "nil SourceContextDiscovery panics",
			In: In{
				Filter: &sous.ResolveFilter{},
			},
			ExpectPanic: true,
		},
		{
			Desc: "no repo specified results in error",
			In: In{
				Discovered: &SourceContextDiscovery{},
			},
			ExpectErr: "no repo specified, please use -repo or run sous inside a git repo with a configured remote",
		},
		{
			Desc: "no repo specified results in error",
			In: In{
				Discovered: &SourceContextDiscovery{},
			},
			ExpectErr: "no repo specified, please use -repo or run sous inside a git repo with a configured remote",
		},
		{
			Desc: "detected repo results in success",
			In: In{
				Discovered: &SourceContextDiscovery{
					SourceContext: &sous.SourceContext{
						PrimaryRemoteURL: "github.com/a/b",
					},
				},
			},
			Expect: func(rrf *RefinedResolveFilter) error {
				expectedRepo := "github.com/a/b"
				if rrf.Repo.All() {
					return fmt.Errorf("got ALL; want %q", expectedRepo)
				}
				actual := *rrf.Repo.Match
				if actual != expectedRepo {
					return fmt.Errorf("got %q; want %q", actual, expectedRepo)
				}
				return nil
			},
		},
		{
			Desc: "flag repo overrides detected",
			In: In{
				Discovered: &SourceContextDiscovery{
					SourceContext: &sous.SourceContext{
						PrimaryRemoteURL: "github.com/from/context",
					},
				},
				Filter: &sous.ResolveFilter{
					Repo: sous.NewResolveFieldMatcher("github.com/from/flags"),
				},
			},
			Expect: func(rrf *RefinedResolveFilter) error {
				expectedRepo := "github.com/from/flags"
				if rrf.Repo.All() {
					return fmt.Errorf("got ALL; want %q", expectedRepo)
				}
				actual := *rrf.Repo.Match
				if actual != expectedRepo {
					return fmt.Errorf("got %q; want %q", actual, expectedRepo)
				}
				return nil
			},
		},
		{
			Desc: "flag offset overrides detected offset",
			In: In{
				Discovered: &SourceContextDiscovery{
					SourceContext: &sous.SourceContext{
						PrimaryRemoteURL: "github.com/from/context",
						OffsetDir:        "from/context",
					},
				},
				Filter: &sous.ResolveFilter{
					Offset: sous.NewResolveFieldMatcher("from/flags"),
				},
			},
			Expect: func(rrf *RefinedResolveFilter) error {
				expectedOffset := "from/flags"
				if rrf.Offset.All() {
					return fmt.Errorf("got ALL; want %q", expectedOffset)
				}
				actual := *rrf.Offset.Match
				if actual != expectedOffset {
					return fmt.Errorf("got %q; want %q", actual, expectedOffset)
				}
				return nil
			},
		},
		{
			Desc: "flag repo sets detected offset to empty",
			In: In{
				Discovered: &SourceContextDiscovery{
					SourceContext: &sous.SourceContext{
						PrimaryRemoteURL: "github.com/from/context",
						OffsetDir:        "from/context",
					},
				},
				Filter: &sous.ResolveFilter{
					Repo: sous.NewResolveFieldMatcher("github.com/from/flags"),
				},
			},
			Expect: func(rrf *RefinedResolveFilter) error {
				expectedOffset := ""
				if rrf.Offset.All() {
					return fmt.Errorf("got ALL; want %q", expectedOffset)
				}
				actual := *rrf.Offset.Match
				if actual != expectedOffset {
					return fmt.Errorf("got %q; want %q", actual, expectedOffset)
				}
				return nil
			},
		},
		{
			Desc: "flag repo and offset override sets detected",
			In: In{
				Discovered: &SourceContextDiscovery{
					SourceContext: &sous.SourceContext{
						PrimaryRemoteURL: "github.com/from/context",
						OffsetDir:        "from/context",
					},
				},
				Filter: &sous.ResolveFilter{
					Repo:   sous.NewResolveFieldMatcher("github.com/from/flags"),
					Offset: sous.NewResolveFieldMatcher("from/flags"),
				},
			},
			Expect: func(rrf *RefinedResolveFilter) error {
				expectedRepo := "github.com/from/flags"
				if rrf.Repo.All() {
					return fmt.Errorf("got ALL; want %q", expectedRepo)
				}
				actualRepo := *rrf.Repo.Match
				if actualRepo != expectedRepo {
					return fmt.Errorf("got %q; want %q", actualRepo, expectedRepo)
				}
				expectedOffset := "from/flags"
				if rrf.Offset.All() {
					return fmt.Errorf("got ALL; want %q", expectedOffset)
				}
				actual := *rrf.Offset.Match
				if actual != expectedOffset {
					return fmt.Errorf("got %q; want %q", actual, expectedOffset)
				}
				return nil
			},
		},
	}

	for _, test := range rrfTests {
		t.Run("", func(t *testing.T) {
			t.Run(test.Desc, func(t *testing.T) {
				t.Parallel()
				if !test.ExpectPanic && test.ExpectErr == "" && test.Expect == nil {
					t.Fatalf("test case must have ExpectPanic, ExpectErr or Expect")
				}
				var recovered interface{}
				var actual *RefinedResolveFilter
				var err error
				func() {
					defer func() { recovered = recover() }()
					actual, err = newRefinedResolveFilter(test.In.Filter, test.In.Discovered)
				}()
				if recovered != nil && !test.ExpectPanic {
					t.Fatalf("got panic (%T): %+v", recovered, recovered)
				}
				if test.ExpectPanic && recovered == nil {
					t.Fatal("did not panic; want panic")
				}
				if test.ExpectErr != "" && err == nil {
					t.Fatalf("got nil error; want %q", test.ExpectErr)
				}
				if test.ExpectErr == "" && err != nil {
					t.Fatalf("got error %q", err.Error())
				}
				if test.ExpectErr != "" && err != nil {
					actualErr := err.Error()
					if test.ExpectErr != actualErr {
						t.Fatalf("got error %q; want %q", actualErr, test.ExpectErr)
					}
				}
				if test.Expect == nil {
					return
				}
				if err := test.Expect(actual); err != nil {
					t.Fatal(err)
				}
			})
		})
	}

}
