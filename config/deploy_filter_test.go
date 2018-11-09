package config

import (
	"testing"

	"github.com/opentable/sous/ext/github"
	sous "github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestDeployFilter(t *testing.T) {
	shc := sous.SourceHostChooser{SourceHosts: []sous.SourceHost{github.SourceHost{}}}

	dep := func(repo, offset, flavor string) *sous.Deployment {
		return &sous.Deployment{
			SourceID: sous.SourceID{
				Location: sous.SourceLocation{
					Repo: repo,
					Dir:  offset,
				},
			},
			Flavor: flavor,
		}
	}

	deploys := []*sous.Deployment{
		dep("github.com/opentable/example", "", ""),
		dep("github.com/opentable/other", "", ""),
		dep("github.com/opentable/example", "somewhere", ""),
		dep("github.com/opentable/flavored", "", "choc"),
		dep("github.com/opentable/flavored", "", "van"),
	}

	testFilter := func(df DeployFilterFlags, idxs ...int) {
		rf, err := df.BuildFilter(shc.ParseSourceLocation)
		assert.NoError(t, err, "For %#v", df)

		for n, dep := range deploys {
			if len(idxs) > 0 && idxs[0] == n {
				assert.True(t, rf.FilterDeployment(dep), "%v doesn't match #%d %v", rf, n, dep)
				idxs = idxs[1:]
			} else {
				assert.False(t, rf.FilterDeployment(dep), "%v matches #%d %v", rf, n, dep)
			}
		}
	}

	testFilter(DeployFilterFlags{All: true}, 0, 1, 2, 3, 4)

	testFilter(DeployFilterFlags{DeploymentIDFlags: DeploymentIDFlags{ManifestIDFlags: ManifestIDFlags{
		SourceLocationFlags: SourceLocationFlags{
			Repo: deploys[0].SourceID.Location.Repo,
		}}}}, 0)

	testFilter(DeployFilterFlags{DeploymentIDFlags: DeploymentIDFlags{ManifestIDFlags: ManifestIDFlags{
		SourceLocationFlags: SourceLocationFlags{
			Repo: deploys[1].SourceID.Location.Repo,
		}}}}, 1)

	testFilter(DeployFilterFlags{}, 0, 1)

	testFilter(DeployFilterFlags{DeploymentIDFlags: DeploymentIDFlags{ManifestIDFlags: ManifestIDFlags{
		SourceLocationFlags: SourceLocationFlags{
			Offset: "",
		}}}}, 0, 1)

	testFilter(DeployFilterFlags{DeploymentIDFlags: DeploymentIDFlags{ManifestIDFlags: ManifestIDFlags{
		SourceLocationFlags: SourceLocationFlags{
			Offset: "*",
		}}}}, 0, 1, 2)

	testFilter(DeployFilterFlags{DeploymentIDFlags: DeploymentIDFlags{ManifestIDFlags: ManifestIDFlags{
		SourceLocationFlags: SourceLocationFlags{
			Offset: "*",
		},
		Flavor: "*",
	}}}, 0, 1, 2, 3, 4)

	testFilter(DeployFilterFlags{DeploymentIDFlags: DeploymentIDFlags{ManifestIDFlags: ManifestIDFlags{
		Flavor: "choc",
	}}}, 3)
}

func TestMakeDeployFilterFlags(t *testing.T) {
	got := MakeDeployFilterFlags(func(*DeployFilterFlags) {})
	want := DeployFilterFlags{}
	if got != want {
		t.Errorf("got noop -> % #v; want % #v", got, want)
	}

	got = MakeDeployFilterFlags(func(dff *DeployFilterFlags) {
		dff.Tag = "1.2.3"
	})
	want = DeployFilterFlags{SourceVersionFlags: SourceVersionFlags{Tag: "1.2.3"}}
	if got != want {
		t.Errorf("got set Tag -> % #v; want % #v", got, want)
	}
}

func TestSourceIDFlags_DeployFilterFlags(t *testing.T) {

	t.Run("empties", func(t *testing.T) {
		in := SourceIDFlags{}
		got := in.DeployFilterFlags()
		want := DeployFilterFlags{}
		if got != want {
			t.Errorf("got %v.SourceIDFlags() == %v; want %v", in, got, want)
		}
	})

	t.Run("repo", func(t *testing.T) {
		in := SourceIDFlags{SourceLocationFlags: SourceLocationFlags{Repo: "repo1"}}
		got := in.DeployFilterFlags()
		want := DeployFilterFlags{
			DeploymentIDFlags: DeploymentIDFlags{
				ManifestIDFlags: ManifestIDFlags{
					SourceLocationFlags: SourceLocationFlags{
						Repo: "repo1",
					},
				},
			},
		}
		if got != want {
			t.Errorf("got %v.SourceIDFlags() == %v; want %v", in, got, want)
		}
	})

	t.Run("tag", func(t *testing.T) {
		in := SourceIDFlags{SourceVersionFlags: SourceVersionFlags{Tag: "1.2.3"}}
		got := in.DeployFilterFlags()
		want := DeployFilterFlags{
			SourceVersionFlags: SourceVersionFlags{Tag: "1.2.3"},
		}
		if got != want {
			t.Errorf("got %v.SourceIDFlags() == %v; want %v", in, got, want)
		}
	})
}

func TestSourceIDFlags_SourceID(t *testing.T) {
	t.Run("empty_semver_error", func(t *testing.T) {
		in := SourceIDFlags{}
		_, gotErr := in.SourceID()
		if gotErr == nil {
			t.Fatalf("got nil error; want not nil")
		}
	})
	t.Run("semver_error", func(t *testing.T) {
		in := SourceIDFlags{SourceVersionFlags: SourceVersionFlags{Tag: "notsemver"}}
		_, gotErr := in.SourceID()
		if gotErr == nil {
			t.Fatalf("got nil error; want not nil")
		}
	})
	t.Run("full", func(t *testing.T) {
		in := SourceIDFlags{
			SourceLocationFlags: SourceLocationFlags{
				Repo:   "repo1",
				Offset: "offset1",
			},
			SourceVersionFlags: SourceVersionFlags{
				Tag:      "1.2.3",
				Revision: "revision1",
			},
		}
		got, err := in.SourceID()
		want := sous.SourceID{
			Location: sous.SourceLocation{
				Repo: "repo1",
				Dir:  "offset1",
			},
			Version: semv.MustParse("1.2.3"),
		}
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("got %v.SourceID() == %v; want %v", in, got, want)
		}
	})
}
