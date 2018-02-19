package server

import (
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/firsterr"
	"github.com/opentable/sous/util/restful"
)

func manifestIDFromValues(qv restful.QueryValues) (sous.ManifestID, error) {
	var r, o, f string
	var err error
	err = firsterr.Returned(
		func() error { r, err = qv.Single("repo"); return err },
		func() error { o, err = qv.Single("offset", ""); return err },
		func() error { f, err = qv.Single("flavor", ""); return err },
	)
	if err != nil {
		return sous.ManifestID{}, err
	}

	return sous.ManifestID{
		Source: sous.SourceLocation{
			Repo: r,
			Dir:  o,
		},
		Flavor: f,
	}, nil
}

func deploymentIDFromValues(qv restful.QueryValues) (sous.DeploymentID, error) {
	cluster, err := qv.Single("cluster")
	if err != nil {
		return sous.DeploymentID{}, err
	}
	mid, err := manifestIDFromValues(qv)
	if err != nil {
		return sous.DeploymentID{}, err
	}
	return sous.DeploymentID{
		ManifestID: mid,
		Cluster:    cluster,
	}, nil
}
