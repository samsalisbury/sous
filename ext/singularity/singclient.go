package singularity

import (
	"github.com/nyarly/spies"
	"github.com/opentable/go-singularity/dtos"
)

type (
	// singClient abstracts the queries we use to retrieve data from singularity.
	singClient interface {
		GetRequest(reqID string, useCache bool) (*dtos.SingularityRequestParent, error)
		GetRequests(useCache bool) (dtos.SingularityRequestParentList, error)
		GetDeploy(reqID, depID string) (*dtos.SingularityDeployHistory, error)
		GetDeploys(reqID string, count int32, page int32) (dtos.SingularityDeployHistoryList, error)
		GetPendingDeploys() (dtos.SingularityPendingDeployList, error)
	}

	singClientSpy struct {
		spy *spies.Spy
	}

	singClientSpyController struct {
		*spies.Spy
	}
)

func newSingClientSpy() (singClientSpy, singClientSpyController) {
	spy := spies.NewSpy()
	return singClientSpy{spy: spy}, singClientSpyController{Spy: spy}
}

func (spy singClientSpy) GetRequest(reqID string, useCache bool) (*dtos.SingularityRequestParent, error) {
	res := spy.spy.Called(reqID, useCache)
	return res.Get(0).(*dtos.SingularityRequestParent), res.Error(1)
}

func (spy singClientSpy) GetRequests(useCache bool) (dtos.SingularityRequestParentList, error) {
	res := spy.spy.Called(useCache)
	return res.Get(0).(dtos.SingularityRequestParentList), res.Error(1)
}

func (spy singClientSpy) GetDeploy(reqID, depID string) (*dtos.SingularityDeployHistory, error) {
	res := spy.spy.Called(reqID, depID)
	return res.Get(0).(*dtos.SingularityDeployHistory), res.Error(1)
}

func (spy singClientSpy) GetDeploys(reqID string, count int32, page int32) (dtos.SingularityDeployHistoryList, error) {
	res := spy.spy.Called(reqID, count, page)
	return res.Get(0).(dtos.SingularityDeployHistoryList), res.Error(1)
}

func (spy singClientSpy) GetPendingDeploys() (dtos.SingularityPendingDeployList, error) {
	res := spy.spy.Called()
	return res.Get(0).(dtos.SingularityPendingDeployList), res.Error(1)
}

func (ctrl singClientSpyController) cannedRequest(answer *dtos.SingularityRequestParent) {
	ctrl.MatchMethod("GetRequest", spies.AnyArgs, answer, nil)
	ctrl.MatchMethod("GetRequests", spies.AnyArgs, dtos.SingularityRequestParentList{answer}, nil)
}

func (ctrl singClientSpyController) cannedDeploy(cannedAnswer *dtos.SingularityDeployHistory) {
	ctrl.MatchMethod("GetDeploy", spies.AnyArgs, cannedAnswer, nil)
	ctrl.MatchMethod("GetDeploys", spies.AnyArgs, dtos.SingularityDeployHistoryList{cannedAnswer}, nil)
}

func (ctrl singClientSpyController) cannedPendingDeploys(cannedAnswer *dtos.SingularityPendingDeployList) {
	ctrl.MatchMethod("GetPendingDeploys", spies.AnyArgs, cannedAnswer)
}
