package sous

import (
	"log"
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

/* TESTS BEGIN */

func TestModifyScale(t *testing.T) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	assert := assert.New(t)
	pair := &DeploymentPair{
		prior: &Deployment{
			SourceVersion: SourceVersion{
				RepoURL: RepoURL("reqid"),
			},
			DeployConfig: DeployConfig{
				NumInstances: 12,
			},
			Cluster: "cluster",
		},
		post: &Deployment{
			SourceVersion: SourceVersion{
				RepoURL: RepoURL("reqid"),
			},
			DeployConfig: DeployConfig{
				NumInstances: 24,
			},
			Cluster: "cluster",
		},
	}

	chanset := NewDiffChans(1)
	nc := NewDummyNameCache()
	client := NewDummyRectificationClient(nc)

	errs := Rectify(chanset, client)
	chanset.Modified <- pair
	chanset.Close()
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
	pair := &DeploymentPair{
		prior: &Deployment{
			SourceVersion: SourceVersion{
				RepoURL: RepoURL("reqid"),
				Version: before,
			},
			DeployConfig: DeployConfig{
				NumInstances: 1,
			},
			Cluster: "cluster",
		},
		post: &Deployment{
			SourceVersion: SourceVersion{
				RepoURL: RepoURL("reqid"),
				Version: after,
			},
			DeployConfig: DeployConfig{
				NumInstances: 1,
			},
			Cluster: "cluster",
		},
	}

	chanset := NewDiffChans(1)

	nc := NewDummyNameCache()
	client := NewDummyRectificationClient(nc)

	errs := Rectify(chanset, client)
	chanset.Modified <- pair
	chanset.Close()
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
	pair := &DeploymentPair{
		prior: &Deployment{
			SourceVersion: SourceVersion{
				RepoURL: RepoURL("reqid"),
				Version: version,
			},
			DeployConfig: DeployConfig{
				NumInstances: 1,
				Resources: Resources{
					"memory": "100",
				},
			},
			Cluster: "cluster",
		},
		post: &Deployment{
			SourceVersion: SourceVersion{
				RepoURL: RepoURL("reqid"),
				Version: version,
			},
			DeployConfig: DeployConfig{
				NumInstances: 1,
				Resources: Resources{
					"memory": "500",
				},
			},
			Cluster: "cluster",
		},
	}

	chanset := NewDiffChans(1)
	nc := NewDummyNameCache()
	client := NewDummyRectificationClient(nc)

	errs := Rectify(chanset, client)
	chanset.Modified <- pair
	chanset.Close()
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
	assert := assert.New(t)
	before, _ := semv.Parse("1.2.3-test")
	after, _ := semv.Parse("2.3.4-new")
	pair := &DeploymentPair{
		prior: &Deployment{
			SourceVersion: SourceVersion{
				RepoURL: RepoURL("reqid"),
				Version: before,
			},
			DeployConfig: DeployConfig{
				NumInstances: 1,
			},
			Cluster: "cluster",
		},
		post: &Deployment{
			SourceVersion: SourceVersion{
				RepoURL: RepoURL("reqid"),
				Version: after,
			},
			DeployConfig: DeployConfig{
				NumInstances: 24,
			},
			Cluster: "cluster",
		},
	}

	chanset := NewDiffChans(1)
	nc := NewDummyNameCache()
	client := NewDummyRectificationClient(nc)

	errs := Rectify(chanset, client)
	chanset.Modified <- pair
	chanset.Close()
	for e := range errs {
		t.Error(e)
	}

	assert.Len(client.created, 0)

	if assert.Len(client.deployed, 1) {
		assert.Regexp("2.3.4", client.deployed[0].imageName)
	}

	if assert.Len(client.scaled, 1) {
		assert.Equal(24, client.scaled[0].count)
	}
}

func TestDeletes(t *testing.T) {
	assert := assert.New(t)

	deleted := &Deployment{
		SourceVersion: SourceVersion{
			RepoURL: RepoURL("reqid"),
		},
		DeployConfig: DeployConfig{
			NumInstances: 12,
		},
		Cluster: "cluster",
	}

	chanset := NewDiffChans(1)
	nc := NewDummyNameCache()
	client := NewDummyRectificationClient(nc)

	errs := Rectify(chanset, client)
	chanset.Deleted <- deleted
	chanset.Close()
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

	chanset := NewDiffChans(1)
	nc := NewDummyNameCache()
	client := NewDummyRectificationClient(nc)

	errs := Rectify(chanset, client)

	created := &Deployment{
		SourceVersion: SourceVersion{
			RepoURL: RepoURL("reqid"),
		},
		DeployConfig: DeployConfig{
			NumInstances: 12,
		},
		Cluster: "cluster",
	}

	chanset.Created <- created

	chanset.Close()

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
