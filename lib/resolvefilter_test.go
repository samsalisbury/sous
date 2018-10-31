package sous

import "testing"

func TestGetSourceID(t *testing.T) {
	zMid := ManifestID{}
	rMid := MustParseManifestID("github.com/blah/blah")

	t.Run("no tag", func(t *testing.T) {
		sid, err := (&ResolveFilter{Tag: ResolveFieldMatcher{}}).SourceID(zMid)
		if err.Error() != `you must provide the -tag flag` {
			t.Errorf("expected error about tag flag, got: %q", err)
		}
		zSid := SourceID{}
		if sid != zSid {
			t.Errorf("non-zero sourceID returned in error case: %v", sid)
		}
	})

	t.Run("badly formatted version tag", func(t *testing.T) {

		sid, err := (&ResolveFilter{Tag: NewResolveFieldMatcher("nope")}).SourceID(zMid)
		if err.Error() != `unexpected character 'n' at position 0` {
			t.Errorf("expected version formatting error, got: %q", err)
		}
		zSid := SourceID{}
		if sid != zSid {
			t.Errorf("non-zero sourceID returned in error case: %v", sid)
		}
	})

	t.Run("successful source id", func(t *testing.T) {
		sid, err := (&ResolveFilter{Cluster: NewResolveFieldMatcher("blah"), Tag: NewResolveFieldMatcher("1.1.1")}).SourceID(rMid)
		if err != nil {
			t.Fatalf("no error expected, but got: %q", err)
		}
		expected := MustParseSourceID("github.com/blah/blah,1.1.1")
		if sid != expected {
			t.Errorf("expected source id like %v got %v", expected, sid)
		}
	})
}

func TestGetDeployID(t *testing.T) {
	zMid := ManifestID{}
	rMid := MustParseManifestID("github.com/blah/blah")

	t.Run("no cluster", func(t *testing.T) {
		did, err := (&ResolveFilter{}).DeploymentID(zMid)
		if err.Error() != `you must select a cluster using the -cluster flag` {
			t.Errorf("expected error about cluster flag, got: %q", err)
		}

		zDid := DeploymentID{}
		if did != zDid {
			t.Errorf("non-zero deploymentID returned in error case: %v", did)
		}
	})

	t.Run("successful deploy id", func(t *testing.T) {
		did, err := (&ResolveFilter{Cluster: NewResolveFieldMatcher("blah"), Tag: NewResolveFieldMatcher("1.1.1")}).DeploymentID(rMid)
		if err != nil {
			t.Fatalf("no error expected, but got: %q", err)
		}
		if did.Cluster != "blah" {
			t.Errorf("expected cluster: blah, got %q", did.Cluster)
		}
		if did.ManifestID != rMid {
			t.Errorf("expected manifest id %v got %v", rMid, did.ManifestID)
		}
	})
}

func TestResolveFilter_String(t *testing.T) {
	cases := []struct {
		in   *ResolveFilter
		want string
	}{
		{
			&ResolveFilter{},
			"<cluster:* repo:* offset:* flavor:* tag:* revision:*>",
		},
		{
			&ResolveFilter{
				Cluster: NewResolveFieldMatcher("1"),
			},
			"<cluster:1 repo:* offset:* flavor:* tag:* revision:*>",
		},
		{
			&ResolveFilter{
				Repo: NewResolveFieldMatcher("1"),
			},
			"<cluster:* repo:1 offset:* flavor:* tag:* revision:*>",
		},
		{
			&ResolveFilter{
				Offset: NewResolveFieldMatcher("1"),
			},
			"<cluster:* repo:* offset:1 flavor:* tag:* revision:*>",
		},
		{
			&ResolveFilter{
				Flavor: NewResolveFieldMatcher("1"),
			},
			"<cluster:* repo:* offset:* flavor:1 tag:* revision:*>",
		},
		{
			&ResolveFilter{
				Tag: NewResolveFieldMatcher("1"),
			},
			"<cluster:* repo:* offset:* flavor:* tag:1 revision:*>",
		},
		{
			&ResolveFilter{
				Revision: NewResolveFieldMatcher("1"),
			},
			"<cluster:* repo:* offset:* flavor:* tag:* revision:1>",
		},
		{
			&ResolveFilter{
				Cluster:  NewResolveFieldMatcher("1"),
				Repo:     NewResolveFieldMatcher("2"),
				Offset:   NewResolveFieldMatcher("3"),
				Flavor:   NewResolveFieldMatcher("4"),
				Tag:      NewResolveFieldMatcher("5"),
				Revision: NewResolveFieldMatcher("6"),
			},
			"<cluster:1 repo:2 offset:3 flavor:4 tag:5 revision:6>",
		},
	}

	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			got := tc.in.String()
			if got != tc.want {
				t.Errorf("got %q; want %q", got, tc.want)
			}
		})
	}
}
