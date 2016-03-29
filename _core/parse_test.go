package core

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestMergeValidConfig(t *testing.T) {
	stateDir := "_testdata/valid_config"
	state, err := Parse(stateDir)
	assertErrNil(t, err)
	assertErrNil(t, state.Validate())
	merged, err := state.Merge()
	assertErrNil(t, err)
	mergedApp := merged.Manifests["github.com/someuser/somerepo"]
	apDeploy := mergedApp.Deployments["asia-pacific"]
	naDeploy := mergedApp.Deployments["north-america"]
	Convey("Merged config", t, func() {
		So(apDeploy.Instance.Count, ShouldEqual, 3)
		So(naDeploy.Instance.Count, ShouldEqual, 2)
		So(apDeploy.Environment["SINGULARITY_URL"], ShouldEqual,
			"http://singularity-asia-pacific.company.com")
		So(naDeploy.Environment["SINGULARITY_URL"], ShouldEqual,
			"http://singularity-north-america.company.com")
	})
	//y, err := yaml.Marshal(merged)
	//assertErrNil(t, err)
	//fmt.Println(string(y))
}

func assertErrNil(t *testing.T, err error) {
	if err != nil {
		t.Fatalf("Got err=\n\t%s\n want nil", err)
	}
}
