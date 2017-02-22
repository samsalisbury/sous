package singularity

import (
	"sort"

	"github.com/opentable/go-singularity/dtos"
)

type testDeployHistoryList map[string]*testDeployHistory

type byTimestamp dtos.SingularityDeployHistoryList

func (b byTimestamp) Len() int      { return len(b) }
func (b byTimestamp) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byTimestamp) Less(i, j int) bool {
	return b[i].DeployMarker.Timestamp < b[j].DeployMarker.Timestamp
}

func (hl testDeployHistoryList) SingularityDeployHistoryList() dtos.SingularityDeployHistoryList {
	var list = make(dtos.SingularityDeployHistoryList, len(hl))
	i := 0
	for _, testDeployHistory := range hl {
		list[i] = testDeployHistory.DeployHistoryItem
		i++
	}
	sort.Sort(byTimestamp(list))
	return list
}
