package singularity

import (
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/opentable/sous/lib"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

/* TESTS BEGIN */

func TestModifyScale(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	assert := assert.New(t)
	pair := &sous.DeploymentPair{
		Prior: &sous.Deployment{
			SourceVersion: sous.SourceVersion{
				RepoURL: sous.RepoURL("reqid"),
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 12,
			},
			Cluster: "cluster",
		},
		Post: &sous.Deployment{
			SourceVersion: sous.SourceVersion{
				RepoURL: sous.RepoURL("reqid"),
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 24,
			},
			Cluster: "cluster",
		},
	}

	mods := make(chan *sous.DeploymentPair, 1)
	errs := make(chan sous.RectificationError)

	nc := NewDummyRegistry()
	client := NewDummyRectificationClient(nc)

	deployer := NewDeployer(nc, client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.deployed, 0)
	assert.Len(client.created, 0)

	if assert.Len(client.scaled, 1) {
		assert.Equal(24, client.scaled[0].count)
	}
}

func TestModifyImage(t *testing.T) {
	assert := assert.New(t)
	before, _ := semv.Parse("1.2.3-test")
	after, _ := semv.Parse("2.3.4-new")
	pair := &sous.DeploymentPair{
		Prior: &sous.Deployment{
			SourceVersion: sous.SourceVersion{
				RepoURL: sous.RepoURL("reqid"),
				Version: before,
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
			},
			Cluster: "cluster",
		},
		Post: &sous.Deployment{
			SourceVersion: sous.SourceVersion{
				RepoURL: sous.RepoURL("reqid"),
				Version: after,
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
			},
			Cluster: "cluster",
		},
	}

	mods := make(chan *sous.DeploymentPair, 1)
	errs := make(chan sous.RectificationError)

	nc := NewDummyRegistry()
	client := NewDummyRectificationClient(nc)
	deployer := NewDeployer(nc, client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.created, 0)
	assert.Len(client.scaled, 0)

	if assert.Len(client.deployed, 1) {
		assert.Regexp("2.3.4", client.deployed[0].imageName)
	}
}

func TestModifyResources(t *testing.T) {
	assert := assert.New(t)
	version := semv.MustParse("1.2.3-test")
	pair := &sous.DeploymentPair{
		Prior: &sous.Deployment{
			SourceVersion: sous.SourceVersion{
				RepoURL: sous.RepoURL("reqid"),
				Version: version,
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
				Resources: sous.Resources{
					"memory": "100",
				},
			},
			Cluster: "cluster",
		},
		Post: &sous.Deployment{
			SourceVersion: sous.SourceVersion{
				RepoURL: sous.RepoURL("reqid"),
				Version: version,
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
				Resources: sous.Resources{
					"memory": "500",
				},
			},
			Cluster: "cluster",
		},
	}

	mods := make(chan *sous.DeploymentPair, 1)
	errs := make(chan sous.RectificationError)

	nc := NewDummyRegistry()
	client := NewDummyRectificationClient(nc)
	deployer := NewDeployer(nc, client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.created, 0)

	if assert.Len(client.deployed, 1) {
		assert.Regexp("1.2.3", client.deployed[0].imageName)
		assert.Regexp("500", client.deployed[0].res["memory"])
	}
}

func TestModify(t *testing.T) {
	Log.Debug.SetOutput(os.Stderr)
	defer Log.Debug.SetOutput(ioutil.Discard)
	assert := assert.New(t)
	before := semv.MustParse("1.2.3-test")
	after := semv.MustParse("2.3.4-new")
	pair := &sous.DeploymentPair{
		Prior: &sous.Deployment{
			SourceVersion: sous.SourceVersion{
				RepoURL: sous.RepoURL("reqid"),
				Version: before,
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 1,
				Volumes: sous.Volumes{
					&sous.Volume{"host", "container", "RO"},
				},
			},
			Cluster: "cluster",
		},
		Post: &sous.Deployment{
			SourceVersion: sous.SourceVersion{
				RepoURL: sous.RepoURL("reqid"),
				Version: after,
			},
			DeployConfig: sous.DeployConfig{
				NumInstances: 24,
				Volumes: sous.Volumes{
					&sous.Volume{"host", "container", "RW"},
				},
			},
			Cluster: "cluster",
		},
	}

	mods := make(chan *sous.DeploymentPair, 1)
	errs := make(chan sous.RectificationError)

	nc := NewDummyRegistry()
	client := NewDummyRectificationClient(nc)
	deployer := NewDeployer(nc, client)

	mods <- pair
	close(mods)
	deployer.RectifyModifies(mods, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.created, 0)

	if assert.Len(client.deployed, 1) {
		assert.Regexp("2.3.4", client.deployed[0].imageName)
		log.Print(client.deployed[0].vols)
		assert.Equal("RW", string(client.deployed[0].vols[0].Mode))
	}

	if assert.Len(client.scaled, 1) {
		assert.Equal(24, client.scaled[0].count)
	}
}

func TestDeletes(t *testing.T) {
	assert := assert.New(t)

	deleted := &sous.Deployment{
		SourceVersion: sous.SourceVersion{
			RepoURL: sous.RepoURL("reqid"),
		},
		DeployConfig: sous.DeployConfig{
			NumInstances: 12,
		},
		Cluster: "cluster",
	}

	dels := make(chan *sous.Deployment, 1)
	errs := make(chan sous.RectificationError)

	nc := NewDummyRegistry()
	client := NewDummyRectificationClient(nc)
	deployer := NewDeployer(nc, client)

	dels <- deleted
	close(dels)
	deployer.RectifyDeletes(dels, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.deployed, 0)
	assert.Len(client.created, 0)

	if assert.Len(client.deleted, 1) {
		req := client.deleted[0]
		assert.Equal("cluster", req.cluster)
		assert.Equal("reqid", req.reqid)
	}
}

func TestCreates(t *testing.T) {
	assert := assert.New(t)

	created := &sous.Deployment{
		SourceVersion: sous.SourceVersion{
			RepoURL: sous.RepoURL("reqid"),
		},
		DeployConfig: sous.DeployConfig{
			NumInstances: 12,
		},
		Cluster: "cluster",
	}

	crts := make(chan *sous.Deployment, 1)
	errs := make(chan sous.RectificationError)

	nc := NewDummyRegistry()
	client := NewDummyRectificationClient(nc)
	deployer := NewDeployer(nc, client)

	crts <- created
	close(crts)
	deployer.RectifyCreates(crts, errs)
	close(errs)

	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.scaled, 0)
	if assert.Len(client.deployed, 1) {
		dep := client.deployed[0]
		assert.Equal("cluster", dep.cluster)
		assert.Equal("reqid 0.0.0", dep.imageName)
	}

	if assert.Len(client.created, 1) {
		req := client.created[0]
		assert.Equal("cluster", req.cluster)
		assert.Equal("reqid", req.id)
		assert.Equal(12, req.count)
	}
}
