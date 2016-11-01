package singularity

import (
	"testing"

	"github.com/opentable/go-singularity/dtos"
	sous "github.com/opentable/sous/lib"
)

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
