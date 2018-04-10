//+build smoke

package smoke

type TestCluster struct {
	BaseDir      string
	RemoteGDMDir string
	Count        int
	Instances    []*Instance
}
