package singularity

import (
	"sort"

	"github.com/opentable/go-singularity/dtos"
)

type testDeployHistoryList map[string]*testDeployHistory

type byDeployMarkerTimestamp dtos.SingularityDeployHistoryList

func (b byDeployMarkerTimestamp) Len() int      { return len(b) }
func (b byDeployMarkerTimestamp) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byDeployMarkerTimestamp) Less(i, j int) bool {
	return b[i].DeployMarker.Timestamp < b[j].DeployMarker.Timestamp
}

// SingularityDeployHistoryList returns the list, removing the Deploy from each
// SingularityDeployHistory in the list, which is what Singularity does.
func (hl testDeployHistoryList) SingularityDeployHistoryList() dtos.SingularityDeployHistoryList {
	var list = make(dtos.SingularityDeployHistoryList, len(hl))
	i := 0
	for _, testDeployHistory := range hl {
		dh := testDeployHistory.DeployHistoryItem
		// Note only DeployMarker and DeployResult are included.
		// This is important as it reflects Singularity's response.
		list[i] = &dtos.SingularityDeployHistory{
			DeployMarker: dh.DeployMarker,
			DeployResult: dh.DeployResult,
		}
		i++
	}
	// Singularity returns the history with newest deploys first.
	sort.Sort(sort.Reverse(byDeployMarkerTimestamp(list)))
	return list
}
