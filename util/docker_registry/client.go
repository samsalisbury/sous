package docker_registry

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/client"
	"golang.org/x/net/context"
)

type V1Schema struct {
	//ContainerConfig ContainerConfig `json:"container_config"`
	CC        ContainerConfig `json:"container_config""`
	Container string          `json:"container"`
}

type ContainerConfig struct {
	Labels map[string]string
	Cmd    []string
}

type Client struct {
	ctx   context.Context
	xport *http.Transport
}

func NewClient() *Client {
	return &Client{
		ctx:   context.Background(),
		xport: &http.Transport{},
	}
}

func (c *Client) BecomeFoolishlyTrusting() {
	c.xport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
}

func (c *Client) Cancel() {
	//at some point, this might cancel contexts/requests outstanding related to this client
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
func (c *Client) LabelsForTaggedImage(regHost, repositoryName, tag string) (labels map[string]string, err error) {
	registryUrl := fmt.Sprintf("https://%s", regHost)

	name, err := reference.ParseNamed(repositoryName)
	if err != nil {
		return
	}

	rep, err := client.NewRepository(c.ctx, name, registryUrl, c.xport)
	if err != nil {
		return
	}

	manifests, err := rep.Manifests(c.ctx)
	if err != nil {
		return
	}

	mani, err := manifests.Get(c.ctx, digest.Digest(""), distribution.WithTagOption{Tag: tag})
	if err != nil {
		log.Print(err)
		return
	}

	switch mani := mani.(type) {
	case *schema1.SignedManifest:
		history := mani.History
		labels = make(map[string]string)
		for _, v1 := range history {
			var historyEntry V1Schema
			json.Unmarshal([]byte(v1.V1Compatibility), &historyEntry)
			//	log.Print(historyEntry.ContainerConfig.Cmd)

			histLabels := historyEntry.CC.Labels
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

func (c *Client) LabelsForImageName(imageName string) (labels map[string]string, err error) {
	ref, err := reference.ParseNamed(imageName)
	if err != nil {
		return
	}

	switch ref := ref.(type) {
	default:
		err = fmt.Errorf("couldn't parse %s into a tagged image name", imageName)
		return
	case reference.NamedTagged:
		regHost, repName := reference.SplitHostname(ref)
		tag := ref.Tag()
		labels, err = c.LabelsForTaggedImage(regHost, repName, tag)
	}
	return
}
