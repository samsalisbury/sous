package sous

import (
	"github.com/opentable/singularity"
	"github.com/opentable/singularity/dtos"
)

type ActualDeploy struct {
}

type Singularity int //bogus

func GetActualDeploy() (adc *ActualDeploy, err error) {
	var singularities []*singularity.Client
	reqCh := make(chan dtos.SingularityRequestParentList)
	errCh := make(chan error)
	allReqs := make([]*dtos.SingularityRequestParent, 0)
	var reqs []*dtos.SingularityRequestParent

	for _, singularity := range singularities {
		go getReqs(singularity, reqCh, errCh)
	}

	select {
	case reqs = <-reqCh:
		allReqs = append(allReqs, reqs...)
	case err = <-errCh:
		return
	}

	return
}

func getReqs(client *singularity.Client, reqCh chan dtos.SingularityRequestParentList, errCh chan error) {
	requests, err := client.GetRequests()
	if err != nil {
		errCh <- err
		return
	}
	reqCh <- requests

}
