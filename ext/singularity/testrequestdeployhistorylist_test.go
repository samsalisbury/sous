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

func (hl testDeployHistoryList) SingularityDeployHistoryList() dtos.SingularityDeployHistoryList {
	var list = make(dtos.SingularityDeployHistoryList, len(hl))
	i := 0
	for _, testDeployHistory := range hl {
		list[i] = testDeployHistory.DeployHistoryItem
		i++
	}
	// Singularity returns the history with newest deploys first.
	sort.Sort(sort.Reverse(byDeployMarkerTimestamp(list)))
	return list
}
