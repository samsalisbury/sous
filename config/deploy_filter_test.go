package config

import (
	"testing"

	"github.com/opentable/sous/ext/github"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
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

func TestNewSourceIDFlags(t *testing.T) {
	sid, err := sous.NewSourceID("repo1", "offset1", "1")
	if err != nil {
		t.Fatalf("test setup failed: %s", err)
	}
	got := NewSourceIDFlags(sid)
	want := SourceIDFlags{
		SourceLocationFlags: SourceLocationFlags{
			Repo:   "repo1",
			Offset: "offset1",
		},
		SourceVersionFlags: SourceVersionFlags{
			Tag: "1",
		},
	}
	if got != want {
		t.Errorf("got %v; want %v", got, want)
	}
}

func TestDeployFilterFlags_EachField(t *testing.T) {
	f := DeployFilterFlags{}
	f.Cluster = "cluster1"
	f.Flavor = "flavor1"
	f.Offset = "offset1"
	f.Repo = "repo1"
	f.Revision = "revision1"
	f.Tag = "tag1"
	got := map[logging.FieldName]string{}
	f.EachField(func(name logging.FieldName, value interface{}) {
		got[name] = value.(string)
	})
	fn := func(name logging.FieldName, want string) {
		v, ok := got[name]
		if !ok {
			t.Errorf("missing field %q", name)
		} else if v != want {
			t.Errorf("got %s=%q; want %q", name, v, want)
		}
	}
	fn(logging.FilterCluster, "cluster1")
	fn(logging.FilterFlavor, "flavor1")
	fn(logging.FilterOffset, "offset1")
	fn(logging.FilterRepo, "repo1")
	fn(logging.FilterRevision, "revision1")
	fn(logging.FilterTag, "tag1")
}
