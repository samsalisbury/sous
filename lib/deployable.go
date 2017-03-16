package sous

import (
	"github.com/satori/go.uuid"
	"regexp"
	"strings"
)

var illegalDeployIDChars = regexp.MustCompile(`[^a-z|^A-Z|^0-9|^_]`)

// Singularity DeployID must be <50
const maxDeployIDLen = 49

// maxVersionLen needs to account for the separator character
// between the version string and the UUID string.
const maxVersionLen = 31

// A Deployable is the pairing of a Deployment and the resolved image that can
// (or has) be used to deploy it.
type Deployable struct {
	Status DeployStatus
	*Deployment
	*BuildArtifact
}

func (d *Deployable) ComputeDeployID() string {
	var uuidTrunc, versionTrunc string
	uuidEntire := stripDeployID(uuid.NewV4().String())
	versionSansMeta := stripMetadata(d.Deployment.SourceID.Version.String())
	versionEntire := sanitizeDeployID(versionSansMeta)

	if len(versionEntire) > maxVersionLen {
		versionTrunc = versionEntire[0:maxVersionLen]
	} else {
		versionTrunc = versionEntire
	}

	// naiveLen is the length of the truncated Version plus
	// the length of an entire UUID plus the length of a separator
	// character.
	naiveLen := len(versionTrunc) + len(uuidEntire) + 1

	if naiveLen > maxDeployIDLen {
		uuidTrunc = uuidEntire[:maxDeployIDLen-len(versionTrunc)-1]
	} else {
		uuidTrunc = uuidEntire
	}

	return strings.Join([]string{
		versionTrunc,
		uuidTrunc,
	}, "_")
}

// SanitizeDeployID replaces characters forbidden in a Singularity deploy ID
// with underscores.
func sanitizeDeployID(in string) string {
	return illegalDeployIDChars.ReplaceAllString(in, "_")
}

// StripDeployID removes all characters forbidden in a Singularity deployID.
func stripDeployID(in string) string {
	return illegalDeployIDChars.ReplaceAllString(in, "")
}

func stripMetadata(in string) string {
	return strings.Split(in, "+")[0]
}
