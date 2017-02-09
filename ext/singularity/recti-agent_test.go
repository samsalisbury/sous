package singularity

import (
	"testing"

	"github.com/opentable/go-singularity/dtos"
	sous "github.com/opentable/sous/lib"
)

func TestSanitizeDeployID(t *testing.T) {
	logTempl := "Got:%s Expected:%s"
	tbl := make(map[string]string)
	tbl["this-has-dashes.and.dots"] = "this_has_dashes_and_dots"
	tbl["forward/slashes"] = "forward_slashes"

	for in, out := range tbl {
		t.Logf("Sanitizing: %s", in)
		s := SanitizeDeployID(in)
		if s != out {
			t.Fatalf(logTempl, s, out)
		} else {
			t.Logf(logTempl, s, out)
		}
	}
}

func TestStripDeployID(t *testing.T) {
	logTempl := "Got:%s Expected:%s"
	tbl := make(map[string]string)
	tbl["this-has-dashes.and.dots"] = "thishasdashesanddots"
	tbl["forward/slashes"] = "forwardslashes"

	for in, out := range tbl {
		t.Logf("Stripping: %s", in)
		s := StripDeployID(in)
		if s != out {
			t.Fatalf(logTempl, s, out)
		} else {
			t.Logf(logTempl, s, out)
		}
	}
}

func TestDetermineRequestType(t *testing.T) {
	pairs := []struct {
		sous.ManifestKind
		dtos.SingularityRequestRequestType
	}{
		{sous.ManifestKindService, dtos.SingularityRequestRequestTypeSERVICE},
		{sous.ManifestKindWorker, dtos.SingularityRequestRequestTypeWORKER},
		{sous.ManifestKindOnDemand, dtos.SingularityRequestRequestTypeON_DEMAND},
		{sous.ManifestKindScheduled, dtos.SingularityRequestRequestTypeSCHEDULED},
		{sous.ManifestKindOnce, dtos.SingularityRequestRequestTypeRUN_ONCE},
	}

	for _, pair := range pairs {
		srrt, err := determineRequestType(pair.ManifestKind)
		if err != nil {
			t.Errorf("Error from determineRequestType: %v", err)
			continue
		}
		if srrt != pair.SingularityRequestRequestType {
			t.Errorf("Got %v expected %v.", srrt, pair.SingularityRequestRequestType)
		}

		// Should test determineManifestKind, but it's more annoying

	}
}
