package docker_registry

type mdChan chan Metadata

// DummyRegistryClient is a type for use in testing - it supports the Client
// interface, while only returning metadata that are fed to it
type DummyRegistryClient struct {
	mds mdChan
}

// NewDummyClient builds and returns a DummyRegistryClient
func NewDummyClient() *DummyRegistryClient {
	mds := make(mdChan, 10)
	return &DummyRegistryClient{mds}
}

// Cancel fulfills part of Client
func (drc *DummyRegistryClient) Cancel() {}

// BecomeFoolishlyTrusting fulfills part of Client
func (drc *DummyRegistryClient) BecomeFoolishlyTrusting() {}

// GetImageMetadata fulfills part of Client
func (drc *DummyRegistryClient) GetImageMetadata(in, et string) (Metadata, error) {
	return <-drc.mds, nil
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
