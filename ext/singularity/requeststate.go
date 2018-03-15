package singularity

import (
	"context"

	singularity "github.com/opentable/go-singularity"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
)

// RequestState uses reqID to retrieve the current DeployState in Singularity.
func RequestState(
	ctx context.Context,
	reqID string,
	url string,
	client *singularity.Client,
	reg sous.Registry,
	clusters sous.Clusters,
	log logging.LogSink,
) (*sous.DeployState, error) {
	reqParent, err := client.GetRequest(reqID, false) //don't use the web cache
	if err != nil {
		return nil, err
	}

	singReq := SingReq{
		SourceURL: url,
		Sing:      client,
		ReqParent: reqParent,
	}

	tgt, err := BuildDeployment(reg, clusters, singReq, log)

	return &tgt, errors.Wrapf(err, "getting request state")
}
