package singularity

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/nyarly/testify/assert"
	"github.com/nyarly/testify/require"
	"github.com/opentable/sous/lib"
)

/* TESTS BEGIN */

func TestBuildDeployRequest(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	di := "dockerImage"
	rID := "reqID"
	env := sous.Env{"test": "yes"}
	rez := sous.Resources{"cpus": "0.1"}
	vols := sous.Volumes{&sous.Volume{}}

	dr, err := buildDeployRequest(di, env, rez, rID, vols)
	require.NoError(err)
	assert.NotNil(dr)
	assert.Equal(dr.Deploy.RequestId, rID)
}

func TestModifyScale(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	assert := assert.New(t)
	pair := &sous.DeploymentPair{
		Prior: &sous.Deployment{
			SourceID: sous.SourceID{
				Location: sous.SourceLocation{
					Repo: "reqid",
				},
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 12,
			},
			ClusterName: "cluster",
			Cluster: &sous.Cluster{
				BaseURL: "cluster",
			},
		},
		Post: &sous.Deployment{
			SourceID: sous.SourceID{
				Location: sous.SourceLocation{
					Repo: "reqid",
				},
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 24,
			},
			ClusterName: "cluster",
			Cluster: &sous.Cluster{
				BaseURL: "cluster",
			},
		},
	}

	mods := make(chan *sous.DeploymentPair, 1)
	errs := make(chan sous.RectificationError)

	nc := sous.NewDummyRegistry()
	client := sous.NewDummyRectificationClient(nc)

	deployer := NewDeployer(nc, client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.Deployed, 0)
	assert.Len(client.Created, 0)

	if assert.Len(client.Scaled, 1) {
		assert.Equal(24, client.Scaled[0].Count)
	}
}

func TestModifyImage(t *testing.T) {
	assert := assert.New(t)
	before := "1.2.3-test"
	after := "2.3.4-new"
	pair := &sous.DeploymentPair{
		Prior: &sous.Deployment{
			SourceID: sous.MustNewSourceID("reqid", "", before),
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
			},
			ClusterName: "cluster",
			Cluster: &sous.Cluster{
				BaseURL: "cluster",
			},
		},
		Post: &sous.Deployment{
			SourceID: sous.MustNewSourceID("reqid", "", after),
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
			},
			ClusterName: "cluster",
			Cluster: &sous.Cluster{
				BaseURL: "cluster",
			},
		},
	}

	mods := make(chan *sous.DeploymentPair, 1)
	errs := make(chan sous.RectificationError)

	nc := sous.NewDummyRegistry()
	client := sous.NewDummyRectificationClient(nc)
	deployer := NewDeployer(nc, client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.Created, 0)
	assert.Len(client.Scaled, 0)

	if assert.Len(client.Deployed, 1) {
		assert.Regexp("2.3.4", client.Deployed[0].ImageName)
	}
}

func TestModifyResources(t *testing.T) {
	assert := assert.New(t)
	version := "1.2.3-test"
	pair := &sous.DeploymentPair{
		Prior: &sous.Deployment{
			SourceID: sous.MustNewSourceID("reqid", "", version),
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
				Resources: sous.Resources{
					"memory": "100",
				},
			},
			ClusterName: "cluster",
			Cluster: &sous.Cluster{
				BaseURL: "cluster",
			},
		},
		Post: &sous.Deployment{
			SourceID: sous.MustNewSourceID("reqid", "", version),
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
				Resources: sous.Resources{
					"memory": "500",
				},
			},
			ClusterName: "cluster",
			Cluster: &sous.Cluster{
				BaseURL: "cluster",
			},
		},
	}

	mods := make(chan *sous.DeploymentPair, 1)
	errs := make(chan sous.RectificationError)

	nc := sous.NewDummyRegistry()
	client := sous.NewDummyRectificationClient(nc)
	deployer := NewDeployer(nc, client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.Created, 0)

	if assert.Len(client.Deployed, 1) {
		assert.Regexp("1.2.3", client.Deployed[0].ImageName)
		assert.Regexp("500", client.Deployed[0].Res["memory"])
	}
}

func TestModify(t *testing.T) {
	Log.Debug.SetOutput(os.Stderr)
	defer Log.Debug.SetOutput(ioutil.Discard)
	assert := assert.New(t)
	before := "1.2.3-test"
	after := "2.3.4-new"
	pair := &sous.DeploymentPair{
		Prior: &sous.Deployment{
			SourceID: sous.MustNewSourceID("reqid", "", before),
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
				Volumes: sous.Volumes{
					{"host", "container", "RO"},
				},
			},
			ClusterName: "cluster",
			Cluster: &sous.Cluster{
				BaseURL: "cluster",
			},
		},
		Post: &sous.Deployment{
			SourceID: sous.MustNewSourceID("reqid", "", after),
			DeployConfig: sous.DeployConfig{
				NumInstances: 24,
				Volumes: sous.Volumes{
					{"host", "container", "RW"},
				},
			},
			ClusterName: "cluster",
			Cluster: &sous.Cluster{
				BaseURL: "cluster",
			},
		},
	}

	mods := make(chan *sous.DeploymentPair, 1)
	errs := make(chan sous.RectificationError)

	nc := sous.NewDummyRegistry()
	client := sous.NewDummyRectificationClient(nc)
	deployer := NewDeployer(nc, client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.Created, 0)

	if assert.Len(client.Deployed, 1) {
		assert.Regexp("2.3.4", client.Deployed[0].ImageName)
		log.Print(client.Deployed[0].Vols)
		assert.Equal("RW", string(client.Deployed[0].Vols[0].Mode))
	}

	if assert.Len(client.Scaled, 1) {
		assert.Equal(24, client.Scaled[0].Count)
	}
}

func TestDeletes(t *testing.T) {
	assert := assert.New(t)

	deleted := &sous.Deployment{
		SourceID: sous.SourceID{
			Location: sous.SourceLocation{
				Repo: "reqid",
			},
		},
		DeployConfig: sous.DeployConfig{
			NumInstances: 12,
		},
		ClusterName: "",
		Cluster: &sous.Cluster{
			BaseURL: "cluster",
		},
	}

	dels := make(chan *sous.Deployment, 1)
	errs := make(chan sous.RectificationError)

	nc := sous.NewDummyRegistry()
	client := sous.NewDummyRectificationClient(nc)
	deployer := NewDeployer(nc, client)

	dels <- deleted
	close(dels)
	deployer.RectifyDeletes(dels, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.Deployed, 0)
	assert.Len(client.Created, 0)

	if assert.Len(client.Deleted, 1) {
		req := client.Deleted[0]
		assert.Equal("cluster", req.Cluster)
		assert.Equal("reqid::", req.Reqid)
	}
}

func TestCreates(t *testing.T) {
	assert := assert.New(t)

	created := &sous.Deployment{
		SourceID: sous.SourceID{
			Location: sous.SourceLocation{
				Repo: "reqid",
			},
		},
		DeployConfig: sous.DeployConfig{
			NumInstances: 12,
		},
		Cluster:     &sous.Cluster{BaseURL: "cluster"},
		ClusterName: "nick",
	}

	crts := make(chan *sous.Deployment, 1)
	errs := make(chan sous.RectificationError)

	nc := sous.NewDummyRegistry()
	client := sous.NewDummyRectificationClient(nc)
	deployer := NewDeployer(nc, client)

	crts <- created
	close(crts)
	deployer.RectifyCreates(crts, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.Scaled, 0)
	if assert.Len(client.Deployed, 1) {
		dep := client.Deployed[0]
		assert.Equal("cluster", dep.Cluster)
		assert.Equal("reqid,0.0.0", dep.ImageName)
	}

	if assert.Len(client.Created, 1) {
		req := client.Created[0]
		assert.Equal("cluster", req.Cluster)
		assert.Equal("reqid::nick", req.ID)
		assert.Equal(12, req.Count)
	}
}
