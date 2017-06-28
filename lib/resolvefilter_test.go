package sous

import "testing"

func TestGetSourceID(t *testing.T) {
	zMid := ManifestID{}
	rMid := MustParseManifestID("github.com/blah/blah")

	t.Run("no tag", func(t *testing.T) {
		sid, err := (&ResolveFilter{}).SourceID(zMid)
		if err.Error() != `you must provide the -tag flag` {
			t.Errorf("expected error about tag flag, got: %q", err)
		}
		zSid := SourceID{}
		if sid != zSid {
			t.Errorf("non-zero sourceID returned in error case: %v", sid)
		}
	})

	t.Run("badly formatted version tag", func(t *testing.T) {
		sid, err := (&ResolveFilter{Tag: "nope"}).SourceID(zMid)
		if err.Error() != `version "nope" not valid: expected something like [servicename-]1.2.3` {
			t.Errorf("expected version formatting error, got: %q", err)
		}
		zSid := SourceID{}
		if sid != zSid {
			t.Errorf("non-zero sourceID returned in error case: %v", sid)
		}
	})

	t.Run("successful source id", func(t *testing.T) {
		sid, err := (&ResolveFilter{Cluster: "blah", Tag: "1.1.1"}).SourceID(rMid)
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
		did, err := (&ResolveFilter{Cluster: "blah", Tag: "1.1.1"}).DeploymentID(rMid)
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
