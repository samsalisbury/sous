package docker_registry

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/v2"
	"github.com/docker/distribution/registry/client"
	"golang.org/x/net/context"
)

type (
	V1Schema struct {
		//ContainerConfig ContainerConfig `json:"container_config"`
		CC        ContainerConfig `json:"container_config""`
		Container string          `json:"container"`
	}

	ContainerConfig struct {
		Labels map[string]string
		Cmd    []string
	}

	// Client for v2 of the docker registry. Maintains state and accumulates e.g. endpoints to make requests against.
	// Although it's developed in concert with Sous, there's a conscious effort to avoid coupling to Sous concepts like e.g. SourceVersion
	liveClient struct {
		ctx        context.Context
		xport      *http.Transport
		registries map[string]*registry
	}

	Client interface {
		LabelsForImageName(string) (map[string]string, error)
		GetImageMetadata(imageName, etag string) (Metadata, error)
		Cancel()
		BecomeFoolishlyTrusting()
	}

	Metadata struct {
		Labels        map[string]string
		Etag          string
		CanonicalName string
		AllNames      []string
	}
)

func NewClient() Client {
	return &liveClient{
		ctx:        context.Background(),
		xport:      &http.Transport{},
		registries: make(map[string]*registry),
	}
}

// BecomeFoolishlyTrusting instructs the client to cease verifying the certificates of registry hosts. This is a terrible idea and
// this method is slated for removal without notice - do not depend on it.
func (c *liveClient) BecomeFoolishlyTrusting() {
	c.xport.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
}

func (c *liveClient) Cancel() {
	//at some point, this might cancel contexts/requests outstanding related to this client
}

// LabelsForImageName retrieves the labels on a particular container from its ImageName
// At the moment the image name has to include a registry hostname and use a tag to identify the image
func (c *liveClient) LabelsForImageName(imageName string) (labels map[string]string, err error) {
	md, err := c.GetImageMetadata(imageName, "")
	return md.Labels, err
}

// LabelsForEtaggedImageName works like LabelsForImageName, with the additional option to send an etag with the request
func (c *liveClient) GetImageMetadata(imageName string, etag string) (Metadata, error) {
	regHost, ref, err := splitHost(imageName)

	if err != nil {
		return Metadata{}, err
	}

	return c.metadataForImage(regHost, ref, etag)
}

func splitHost(in string) (url string, ref reference.Named, err error) {
	ref, err = reference.ParseNamed(in)
	if err != nil {
		return
	}

	url, name := reference.SplitHostname(ref)
	ref, err = updateName(ref, name)
	return
}

func joinHost(host string, ref reference.Named) (reference.Named, error) {
	if host == "" {
		return ref, nil
	}

	return updateName(ref, strings.Join([]string{host, ref.Name()}, "/"))
}

func updateName(rn reference.Named, name string) (ref reference.Named, err error) {
	nr, err := reference.ParseNamed(name)
	if err != nil {
		return
	}

	switch r := rn.(type) {
	default:
		return nil, fmt.Errorf("Image name has neither tag nor digest")
	case reference.Digested:
		ref, err = reference.WithDigest(nr, r.Digest())
	case reference.Tagged:
		ref, err = reference.WithTag(nr, r.Tag())
	}

	return
}

func digestRef(ref reference.Named, digst string) (reference.Canonical, error) {
	rn, err := reference.ParseNamed(ref.Name())
	if err != nil {
		return nil, err
	}

	d := digest.Digest(digst)

	return reference.WithDigest(rn, d)
}

func (c *liveClient) registryForUrl(url string) (*registry, error) {
	if reg, ok := c.registries[url]; ok {
		return reg, nil
	}
	reg, err := newRegistry(url, c.xport)
	if err != nil {
		return nil, err
	}
	c.registries[url] = reg
	return reg, nil
}

// LabelsForTaggedImage makes a query to a docker registry an returns a map of the labels on that image.
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
func (c *liveClient) metadataForImage(regHost string, ref reference.Named, etag string) (md Metadata, err error) {
	registryUrl := fmt.Sprintf("https://%s", regHost)

	// slightly weird but: a non-empty etag implies that we've seen this
	// digest-named container before - and a digest reference should be
	// immutable.
	if _, ok := ref.(reference.Digested); ok && etag != "" {
		return Metadata{}, distribution.ErrManifestNotModified
	}

	rep, err := c.registryForUrl(registryUrl)
	if err != nil {
		return
	}

	mani, headers, err := rep.getManifestWithEtag(c.ctx, ref, etag)
	if err != nil {
		return
	}

	md = Metadata{
		AllNames: make([]string, 2),
		Labels:   make(map[string]string),
		Etag:     headers.Get("Etag"),
	}
	md.AllNames[0] = ref.String()
	dr, err := digestRef(ref, headers.Get("Docker-Content-Digest"))
	if err == nil {
		md.AllNames[1] = dr.String()
		md.CanonicalName = dr.String()
	}

	switch mani := mani.(type) {
	case *schema1.SignedManifest:
		history := mani.History
		for _, v1 := range history {
			var historyEntry V1Schema
			json.Unmarshal([]byte(v1.V1Compatibility), &historyEntry)
			//	log.Print(historyEntry.ContainerConfig.Cmd)

			histLabels := historyEntry.CC.Labels
			// XXX It's unclear from the docker spec which order the labels appear in.
			// It may be that this is the wrong order to merge labels in -
			// and I have the dim recollection that the order may change between schema 1 vs. 2

			for k, v := range histLabels {
				md.Labels[k] = v
			}
		}
	default:
		err = fmt.Errorf("Cripes! v2 manifest, which is awesome, but we have no idea how to parse it. Contact your nearest sous chef.")
	}

	return
}

// NewRepository creates a new Repository for the given repository name and base URL.
func newRegistry(baseURL string, transport http.RoundTripper) (*registry, error) {
	ub, err := v2.NewURLBuilderFromString(baseURL, false)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		Transport:     transport,
		CheckRedirect: checkHTTPRedirect,
		// TODO(dmcgowan): create cookie jar
	}

	return &registry{
		client: client,
		ub:     ub,
	}, nil
}

type registry struct {
	client *http.Client
	ub     *v2.URLBuilder
}

func (ms *registry) getRequest(ref reference.Named, etag string) (req *http.Request, err error) {
	u, err := ms.ub.BuildManifestURL(ref)
	if err != nil {
		return nil, err
	}

	req, err = http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	for _, t := range distribution.ManifestMediaTypes() {
		req.Header.Add("Accept", t)
	}

	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	return req, nil
}

func (ms *registry) manifestFromResponse(resp *http.Response) (distribution.Manifest, error) {
	if resp.StatusCode == http.StatusNotModified {
		return nil, distribution.ErrManifestNotModified
	} else if client.SuccessStatus(resp.StatusCode) {
		mt := resp.Header.Get("Content-Type")
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return nil, err
		}
		m, _, err := distribution.UnmarshalManifest(mt, body)
		if err != nil {
			return nil, err
		}
		return m, nil
	}
	return nil, client.HandleErrorResponse(resp)
}

func (ms *registry) getManifestWithEtag(ctx context.Context, ref reference.Named, etag string) (distribution.Manifest, http.Header, error) {
	var err error

	req, err := ms.getRequest(ref, etag)
	if err != nil {
		return nil, http.Header{}, err
	}

	resp, err := ms.client.Do(req)
	if err != nil {
		return nil, http.Header{}, err
	}
	defer resp.Body.Close()
	mf, err := ms.manifestFromResponse(resp)
	if err != nil {
		return nil, http.Header{}, err
	}

	return mf, resp.Header, err
}
