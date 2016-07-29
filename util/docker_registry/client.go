package docker_registry

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/docker/distribution"
	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/distribution/registry/api/v2"
	"github.com/docker/distribution/registry/client"
	"github.com/docker/distribution/registry/client/transport"
	"golang.org/x/net/context"
)

type (
	// V1Schema Represents the original v1 schema data for a container
	V1Schema struct {
		//ContainerConfig ContainerConfig `json:"container_config"`
		CC        ContainerConfig `json:"container_config""`
		Container string          `json:"container"`
	}

	// ContainerConfig captures the configuration of a docker container
	ContainerConfig struct {
		Labels map[string]string
		Cmd    []string
	}

	// Client for v2 of the docker registry. Maintains state and accumulates
	// e.g. endpoints to make requests against. Although it's developed in
	// concert with Sous, there's a conscious effort to avoid coupling to Sous
	// concepts like SourceID.
	liveClient struct {
		ctx   context.Context
		xport *http.Transport
		Registries
	}

	// Registries is a map+Mutex
	Registries struct {
		regs map[string]*registry
		sync.Mutex
	}

	// Client is the interface for interacting with a docker registry
	Client interface {
		LabelsForImageName(string) (map[string]string, error)
		GetImageMetadata(imageName, etag string) (Metadata, error)
		AllTags(repoName string) ([]string, error)
		Cancel()
		BecomeFoolishlyTrusting()
	}

	// Metadata represents the descriptive data for a docker image
	Metadata struct {
		Registry      string
		Labels        map[string]string
		Etag          string
		CanonicalName string
		AllNames      []string
	}
)

// NewRegistries makes a Registries
func NewRegistries() Registries {
	return Registries{regs: make(map[string]*registry)}
}

// AddRegistry adds a registry to the registry map
func (rs *Registries) AddRegistry(n string, r *registry) error {
	rs.Lock()
	defer rs.Unlock()
	rs.regs[n] = r
	return nil
}

// GetRegistry gets a registry from the registry map
func (rs *Registries) GetRegistry(n string) *registry {
	return rs.regs[n]
}

// DeleteRegistry deletes a registry from the map
func (rs *Registries) DeleteRegistry(n string) error {
	rs.Lock()
	defer rs.Unlock()
	delete(rs.regs, n)
	return nil
}

// NewClient builds a new client
func NewClient() Client {
	return &liveClient{
		ctx:        context.Background(),
		xport:      &http.Transport{},
		Registries: NewRegistries(),
	}
}

// BecomeFoolishlyTrusting instructs the client to cease verifying the certificates of registry hosts.
// This is a terrible idea and this method is slated for removal without notice - do not depend on it.
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

// AllTags returns a list of tags for a particular repo
func (c *liveClient) AllTags(repoName string) ([]string, error) {
	//log.Printf("AllTags(%s)", repoName)
	regHost, ref, err := splitHost(repoName)
	if err != nil {
		return []string{}, err
	}

	rep, err := c.registryForHostname(regHost)
	if err != nil {
		return []string{}, err
	}

	//log.Printf("Getting tags for %v from %s", ref, regHost)
	return rep.getRepoTags(ref)
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

	//log.Printf("updateName: %#v %#v", rn, nr)

	switch r := rn.(type) {
	default:
		return nil, fmt.Errorf("Image name has neither tag nor digest (%T)", rn)
	case reference.Digested:
		ref, err = reference.WithDigest(nr, r.Digest())
	case reference.Tagged:
		ref, err = reference.WithTag(nr, r.Tag())
	case reference.Named:
		ref, err = nr, err
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

func (c *liveClient) registryForHostname(regHost string) (*registry, error) {
	url := fmt.Sprintf("https://%s", regHost)
	if reg := c.GetRegistry(url); reg != nil {
		return reg, nil
	}
	reg, err := newRegistry(url, c.xport)
	if err != nil {
		return nil, err
	}
	c.AddRegistry(url, reg)
	return reg, nil
}

type stubConfig struct {
	Config stubImage `json:config`
}

type stubImage struct {
	Labels map[string]string
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
	// slightly weird but: a non-empty etag implies that we've seen this
	// digest-named container before - and a digest reference should be
	// immutable.
	if _, ok := ref.(reference.Digested); ok && etag != "" {
		return Metadata{}, distribution.ErrManifestNotModified
	}

	rep, err := c.registryForHostname(regHost)
	if err != nil {
		return
	}

	mani, dg, headers, err := rep.getManifestWithEtag(c.ctx, ref, etag)
	if err != nil {
		return
	}

	md = Metadata{
		Registry: regHost,
		AllNames: make([]string, 2),
		Labels:   make(map[string]string),
		Etag:     headers.Get("Etag"),
	}
	md.AllNames[0] = ref.String()

	md.CanonicalName = ref.Name() + "@" + dg.String()
	md.AllNames[1] = md.CanonicalName

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
	case *schema2.DeserializedManifest:
		var cj []byte
		cj, err = rep.getBlob(c.ctx, ref, mani.Config.Digest)

		var c stubConfig
		err = json.Unmarshal(cj, &c)

		if err != nil {
			return
		}

		md.Labels = c.Config.Labels
	default:
		// We shouldn't receive this, because we shouldn't include the Accept
		// header that would trigger it. To begin work on this (because...?) start
		// by adding schema2 as an import - it's a sibling of schema1. Schema2
		// includes a 'config' key, which has a digest for a blob - see
		// distribution/pull_v2 pullSchema2ImageConfig() (~ ln 677)
		err = fmt.Errorf("Cripes! Don't know that format of manifest")
	}

	return
}

/*
 */

// All returns all tag// NewRepository creates a new Repository for the given repository name and base URL.
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

func (r *registry) getRequest(u, etag string) (req *http.Request, err error) {
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

type tagsResponse struct {
	Tags []string `json:"tags"`
}

func (r *registry) getBlob(ctx context.Context, name reference.Named, dgst digest.Digest) ([]byte, error) {
	ref, err := reference.WithDigest(name, dgst)
	if err != nil {
		return nil, err
	}
	blobURL, err := r.ub.BuildBlobURL(ref)
	if err != nil {
		return nil, err
	}

	reader := transport.NewHTTPReadSeeker(r.client, blobURL,
		func(resp *http.Response) error {
			if resp.StatusCode == http.StatusNotFound {
				return distribution.ErrBlobUnknown
			}
			return client.HandleErrorResponse(resp)
		})
	defer reader.Close()

	return ioutil.ReadAll(reader)
}

func (r *registry) getRepoTags(ref reference.Named) (tags []string, err error) {
	u, err := r.ub.BuildTagsURL(ref)
	if err != nil {
		return tags, err
	}

	req, err := r.getRequest(u, "")
	if err != nil {
		return nil, err
	}

	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if !client.SuccessStatus(resp.StatusCode) {
		log.Printf("Error response to %#v %v", req, req.URL)
		return tags, client.HandleErrorResponse(resp)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return tags, err
	}

	var tr tagsResponse
	if err := json.Unmarshal(b, &tr); err != nil {
		return tags, err
	}
	tags = tr.Tags
	return tags, nil
}

func (r *registry) getManifestWithEtag(ctx context.Context, ref reference.Named, etag string) (mf distribution.Manifest, d digest.Digest, h http.Header, err error) {
	u, err := r.ub.BuildManifestURL(ref)

	if err != nil {
		return
	}

	req, err := r.getRequest(u, etag)
	if err != nil {
		return
	}

	resp, err := r.client.Do(req)
	defer safeCloseBody(resp)

	if err != nil {
		return
	}

	h = resp.Header
	mf, d, err = r.manifestFromResponse(resp)
	return
}

func safeCloseBody(r *http.Response) {
	defer func() { recover() }()
	r.Body.Close()
}

func (r *registry) manifestFromResponse(resp *http.Response) (distribution.Manifest, digest.Digest, error) {
	if resp.StatusCode == http.StatusNotModified {
		return nil, "", distribution.ErrManifestNotModified
	} else if client.SuccessStatus(resp.StatusCode) {
		mt := resp.Header.Get("Content-Type")
		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return nil, "", err
		}
		m, _, err := distribution.UnmarshalManifest(mt, body)
		if err != nil {
			return nil, "", err
		}

		var d digest.Digest
		switch v := m.(type) {
		case *schema1.SignedManifest:
			//log.Print(string(v.Canonical))
			d = digest.FromBytes(v.Canonical)
		case *schema2.DeserializedManifest:
			_, pl, err := m.Payload()
			if err != nil {
				return nil, "", err
			}

			//log.Print(string(pl))
			d = digest.FromBytes(pl)
		default:
			return nil, "", fmt.Errorf("unsupported manifest format")

		}
		//		log.Printf("%T", m)
		//		log.Print("Calced: ", d)
		//		log.Print("Header: ", resp.Header.Get("Docker-Content-Digest"))
		//		log.Print("Docker: sha256:d3d75a393555a8eb6bf1e94736b90b84712638e5f3dbd7728355310dbd4f1684") //docker pull
		return m, d, nil
	}
	return nil, "", client.HandleErrorResponse(resp)
}
