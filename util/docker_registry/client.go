package docker_registry

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"github.com/samsalisbury/semv"
	"golang.org/x/net/context"
)

type V1Schema struct {
	ContainerConfig ContainerConfig `json:container_config`
}

type ContainerConfig struct {
	Labels map[string]string
}

// Makes a query to a docker registry an returns a map of the labels on that image.
// Currently supports the v2.0 registry Schema v1 (not to be confused with Schema v2)
// This shouldn't be a problem, since the second version of the schema isn't due until the summer
// and registries that suport it are supposed to use HTTP Accept headers to negotiate with clients.
// c.f. https://github.com/docker/distribution/blob/master/docs/spec/manifest-v2-1.md
// and  https://github.com/docker/distribution/blob/master/docs/spec/manifest-v2-2.md
//
// labelsForTaggedImage(
//  "http://artifactory.otenv.com/artifactory/api/docker/docker-v2/v2",
//	"demo-server",
//	"demo-server-0.7.3-SNAPSHOT-20160329_202654_teamcity-unconfigured"
// )
// ( which returns an empty map, since the demo-server doesn't have labels... )
func LabelsForTaggedImage(registryUrl, repositoryName, tag string) (labels map[string]string, err error) {

	ctx := context.Background()
	name, err := reference.ParseNamed(repositoryName)
	if err != nil {
		return
	}

	xport := new(http.Transport)
	rep, err := client.NewRepository(ctx, name, registryUrl, xport)
	if err != nil {
		return
	}

	manifests, err := rep.Manifests(ctx)
	if err != nil {
		return
	}

	mani, err := manifests.Get(ctx, digest.Digest(""), distribution.WithTagOption{Tag: tag})
	if err != nil {
		return
	}

	switch mani := mani.(type) {
	case *schema1.SignedManifest:
		history := mani.History
		for _, v1 := range history {
			var historyEntry V1Schema
			json.Unmarshal([]byte(v1.V1Compatibility), &historyEntry)
			histLabels := historyEntry.ContainerConfig.Labels

			// XXX It's unclear from the docker spec which order the labels appear in.
			// It may be that this is the wrong order to merge labels in -
			// and I have the dim recollection that the order may change between schema 1 vs. 2
			for k, v := range histLabels {
				labels[k] = v
			}
		}
	default:
		err = fmt.Errorf("Cripes! v2 manifest, which is awesome, but we have no idea how to parse it. Contact your nearest sous chef.")
	}

	return
}

func LabelsForImageName(imageName string) (repo, path string, version semv.Version, err error) {
	ref, err := reference.ParseNamed(imageName)
	if err != nil {
		return
	}

	var labels map[string]string
	switch ref := ref.(type) {
	default:
		err = fmt.Errorf("couldn't parse %s into a tagged image name", imageName)
		return
	case reference.NamedTagged:
		regUrl, repName := reference.SplitHostname(ref)
		tag := ref.Tag()
		labels, err = LabelsForTaggedImage(regUrl, repName, tag)
	}

	if err != nil {
		return
	}

	missingLabels := make([]string, 0, 3)
	repo, present := labels[DockerRepoLabel]
	if !present {
		missingLabels = append(missingLabels, DockerRepoLabel)
	}

	versionStr, present := labels[DockerVersionLabel]
	if !present {
		missingLabels = append(missingLabels, DockerVersionLabel)
	}

	path, present = labels[DockerPathLabel]
	if !present {
		missingLabels = append(missingLabels, DockerPathLabel)
	}

	if len(missingLabels) > 0 {
		err = fmt.Errorf("Missing labels on manifest for %s: %v", imageName, missingLabels)
		return
	}

	version, err = semv.Parse(versionStr)

	return
}
