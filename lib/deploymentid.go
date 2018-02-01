package sous

import (
	"crypto/md5"
	"io"
)

// A DeploymentID identifies a deployment.
type DeploymentID struct {
	ManifestID ManifestID
	Cluster    string
}

// Digest genrates an MD5 sum generated by the combination of  Git repo,
// project flavor, and the name of the cluster separted by null bytes.
func (did *DeploymentID) Digest() []byte {
	h := md5.New()
	sep := []byte{0}
	io.WriteString(h, did.ManifestID.Source.String())
	h.Write(sep)
	io.WriteString(h, did.ManifestID.Flavor)
	h.Write(sep)
	io.WriteString(h, did.Cluster)

	return h.Sum(nil)
}

func (did DeploymentID) String() string {
	return did.Cluster + ":" + did.ManifestID.String()
}
