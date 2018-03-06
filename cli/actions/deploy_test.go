package actions

import (
	"testing"

	"github.com/opentable/sous/util/restful"
)

func TestDeploy_Do(t *testing.T) {

	fakeClient := restful.HTTPClient

	d := &Deploy{}

	err := d.Do()

}
