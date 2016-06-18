package docker_registry

type mdChan chan Metadata
type tChan chan []string

// DummyRegistryClient is a type for use in testing - it supports the Client
// interface, while only returning metadata that are fed to it
type DummyRegistryClient struct {
	mds mdChan
	ts  tChan
}

// NewDummyClient builds and returns a DummyRegistryClient
func NewDummyClient() *DummyRegistryClient {
	mds := make(mdChan, 10)
	ts := make(tChan, 10)
	return &DummyRegistryClient{mds, ts}
}

// Cancel fulfills part of Client
func (drc *DummyRegistryClient) Cancel() {}

// BecomeFoolishlyTrusting fulfills part of Client
func (drc *DummyRegistryClient) BecomeFoolishlyTrusting() {}

// GetImageMetadata fulfills part of Client
func (drc *DummyRegistryClient) GetImageMetadata(in, et string) (Metadata, error) {
	return <-drc.mds, nil
}

// AllTags fulfills part of Client
func (drc *DummyRegistryClient) AllTags(rn string) ([]string, error) {
	return <-drc.ts, nil
}

// LabelsForImageName fulfills part of Client
func (drc *DummyRegistryClient) LabelsForImageName(in string) (map[string]string, error) {
	md := <-drc.mds
	return md.Labels, nil
}

// FeedMetadata is the strings on the marrionette of DummyRegistryClient -
// having triggered a call to GetImageMetadata or LabelsForImageName, use
// FeedMetadata to send the Metadata that the notional docker
// registry might return
func (drc *DummyRegistryClient) FeedMetadata(md Metadata) {
	drc.mds <- md
}
