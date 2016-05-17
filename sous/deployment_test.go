package sous

import (
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentEqual(t *testing.T) {
	assert := assert.New(t)

	dep := Deployment{}
	assert.True(dep.Equal(Deployment{}))

	other := Deployment{
		Annotation: Annotation{
			RequestId: "somewhere around here",
		},
	}
	assert.True(dep.Equal(other))
}

func TestCannonName(t *testing.T) {
	assert := assert.New(t)

	vers, _ := semv.Parse("1.2.3-test+thing")
	dep := Deployment{
		SourceVersion: SourceVersion{
			RepoURL:    RepoURL("one"),
			RepoOffset: RepoOffset("two"),
			Version:    vers,
		},
	}
	str := dep.SourceVersion.CanonicalName().String()
	assert.Regexp("one", str)
	assert.Regexp("two", str)
}
