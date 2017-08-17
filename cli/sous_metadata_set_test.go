package cli

import (
	"testing"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful/restfultest"
	"github.com/opentable/sous/util/spies"
	"github.com/stretchr/testify/assert"
)

func TestMetadataSet(t *testing.T) {
	cl, control := restfultest.NewHTTPClientSpy()
	mani := testManifest("with-metadata")
	rf := sous.ResolveFilter{
		Repo:    mani.Source.Repo,
		Cluster: "cluster-1",
	}

	sms := &SousMetadataSet{
		TargetManifestID: graph.TargetManifestID(mani.ID()),
		ResolveFilter:    &rf,
		HTTPClient:       graph.HTTPClient{cl},
	}

	updater, upctl := restfultest.NewUpdateSpy()
	control.MatchMethod(
		"Retrieve",
		spies.Once(),
		testManifest("with-metadata"), updater, nil,
	)
	control.Any(
		"Retrieve",
		testManifest("with-metadata"), restfultest.DummyUpdater(), nil,
	)
	upctl.Any(
		"Update",
		nil,
	)

	res := sms.Execute([]string{"BuildBranch", "development"})
	assert.Equal(t, 0, res.ExitCode())

	if assert.Len(t, control.Calls(), 1) {
		args := control.Calls()[0].PassedArgs()
		assert.Regexp(t, "/manifest", args.String(0))
	}
	if assert.Len(t, upctl.Calls(), 1) {
		args := upctl.Calls()[0].PassedArgs()
		orig := testManifest("with-metadata")
		mani := args.Get(0).(*sous.Manifest)
		assert.Equal(t, "development", mani.Deployments["cluster-1"].Metadata["BuildBranch"])
		assert.NotEqual(t,
			orig.Deployments["cluster-1"].Metadata["BuildBranch"],
			mani.Deployments["cluster-1"].Metadata["BuildBranch"],
		)
		assert.Equal(t,
			orig.Deployments["cluster-2"].Metadata["BuildBranch"],
			mani.Deployments["cluster-2"].Metadata["BuildBranch"],
		)
	}
}
