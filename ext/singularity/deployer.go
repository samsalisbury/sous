package singularity

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/opentable/go-singularity"
	"github.com/opentable/sous/lib"
	"github.com/opentable/swaggering"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

// Both of these values are (for reasons only known to the spirits)
// _configurable_ in singularity. If you've done something silly like configure
// them differently than their defaults, at the moment we wish you the best of
// luck, and vaya con Dios.
// c.f. https://github.com/HubSpot/Singularity/blob/master/Docs/reference/configuration.md#limits

// Singularity DeployID must be <50
const maxDeployIDLen = 49

// Singularity RequestID must be <100
const maxRequestIDLen = 99

// maxVersionLen needs to account for the separator character
// between the version string and the UUID string.
const maxVersionLen = 31

/*
The imagined use case here is like this:

intendedSet := getFromManifests()
existingSet := getFromSingularity()

dChans := intendedSet.Diff(existingSet)
*/

type (
	deployer struct {
		Client  rectificationClient
		singFac func(string) *singularity.Client
	}

	// rectificationClient abstracts the raw interactions with Singularity.
	rectificationClient interface {
		// Deploy creates a new deploy on a particular requeust
		Deploy(d sous.Deployable, reqID string) error

		// PostRequest sends a request to a Singularity cluster to initiate
		PostRequest(d sous.Deployable, reqID string) error

		// DeleteRequest instructs Singularity to delete a particular request
		DeleteRequest(cluster, reqID, message string) error
	}

	// DTOMap is shorthand for map[string]interface{}
	dtoMap map[string]interface{}
)

func sanitizeDeployID(in string) string {
	return illegalDeployIDChars.ReplaceAllString(in, "_")
}

func stripDeployID(in string) string {
	return illegalDeployIDChars.ReplaceAllString(in, "")
}

// NewDeployer creates a new Singularity-based sous.Deployer.
func NewDeployer(c rectificationClient) sous.Deployer {
	return &deployer{Client: c}
}

// RectifyCreates implements sous.Deployer on deployer
func (r *deployer) RectifyCreates(cc <-chan *sous.DeployablePair, errs chan<- sous.DiffResolution) {
	for d := range cc {
		result := sous.DiffResolution{DeploymentID: d.ID()}
		if err := r.RectifySingleCreate(d); err != nil {
			result.Desc = "not created"
			switch t := err.(type) {
			default:
				result.Error = sous.WrapResolveError(&sous.CreateError{Deployment: d.Post.Deployment.Clone(), Err: err})
			case *swaggering.ReqError:
				if t.Status == 400 {
					result.Error = sous.WrapResolveError(err)
				} else {
					result.Error = sous.WrapResolveError(&sous.CreateError{Deployment: d.Post.Deployment.Clone(), Err: err})
				}
			}
		} else {
			result.Desc = "created"
		}
		Log.Vomit.Printf("Reporting result of create: %#v", result)
		errs <- result
	}
}

func (r *deployer) SetSingularityFactory(fn func(string) *singularity.Client) {
	r.singFac = fn
}

func (r *deployer) buildSingClient(url string) *singularity.Client {
	if r.singFac == nil {
		return singularity.NewClient(url)
	}
	return r.singFac(url)
}

func rectifyRecover(d interface{}, f string, err *error) {
	if r := recover(); r != nil {
		sous.Log.Warn.Printf("Panic in %s with %# v", f, d)
		sous.Log.Warn.Printf("  %v", r)
		sous.Log.Warn.Print(string(debug.Stack()))
		*err = errors.Errorf("Panicked")
	}
}

func (r *deployer) RectifySingleCreate(d *sous.DeployablePair) (err error) {
	Log.Debug.Printf("Rectifying creation %q:  \n %# v", d.ID(), d.Post)
	defer rectifyRecover(d, "RectifySingleCreate", &err)
	if err != nil {
		return err
	}
	reqID, err := computeRequestID(d.Post)
	if err != nil {
		return err
	}
	if err = r.Client.PostRequest(*d.Post, reqID); err != nil {
		return err
	}
	return r.Client.Deploy(*d.Post, reqID)
}

func (r *deployer) RectifyDeletes(dc <-chan *sous.DeployablePair, errs chan<- sous.DiffResolution) {
	for d := range dc {
		result := sous.DiffResolution{DeploymentID: d.ID()}
		if err := r.RectifySingleDelete(d); err != nil {
			result.Error = sous.WrapResolveError(&sous.DeleteError{Deployment: d.Prior.Deployment.Clone(), Err: err})
			result.Desc = "not deleted"
		} else {
			result.Desc = "deleted"
		}
		Log.Vomit.Printf("Reporting result of delete: %#v", result)
		errs <- result
	}
}

func (r *deployer) RectifySingleDelete(d *sous.DeployablePair) (err error) {
	defer rectifyRecover(d, "RectifySingleDelete", &err)
	data, ok := d.ExecutorData.(*singularityTaskData)
	if !ok {
		return errors.Errorf("Delete record %#v doesn't contain Singularity compatible data: was %T\n\t%#v", d.ID(), data, d)
	}
	requestID := data.requestID

	// TODO: Alert the owner of this request that there is no manifest for it;
	// they should either delete the request manually, or else add the manifest back.
	sous.Log.Warn.Printf("NOT DELETING REQUEST %q (FOR: %q)", requestID, d.ID())
	return nil
	// The following line deletes requests when it is not commented out.
	//return r.Client.DeleteRequest(d.Cluster.BaseURL, requestID, "deleting request for removed manifest")
}

func (r *deployer) RectifyModifies(
	mc <-chan *sous.DeployablePair, errs chan<- sous.DiffResolution) {
	for pair := range mc {
		result := sous.DiffResolution{DeploymentID: pair.ID()}
		if err := r.RectifySingleModification(pair); err != nil {
			dp := &sous.DeploymentPair{
				Prior: pair.Prior.Deployment.Clone(),
				Post:  pair.Post.Deployment.Clone(),
			}
			Log.Debug.Printf("%#v", err)
			result.Error = sous.WrapResolveError(&sous.ChangeError{Deployments: dp, Err: err})
			result.Desc = "not updated"
		} else if pair.Prior.Status == sous.DeployStatusFailed || pair.Post.Status == sous.DeployStatusFailed {
			result.Desc = "updated"
			result.Error = sous.WrapResolveError(&sous.FailedStatusError{})
		} else {
			result.Desc = "updated"
		}
		Log.Vomit.Printf("Reporting result of modify: %#v", result)
		errs <- result
	}
}

func (r *deployer) RectifySingleModification(pair *sous.DeployablePair) (err error) {
	Log.Debug.Printf("Rectifying modified %q: \n  %# v \n    =>  \n  %# v", pair.ID(), pair.Prior.Deployment, pair.Post.Deployment)
	defer rectifyRecover(pair, "RectifySingleModification", &err)

	data, ok := pair.ExecutorData.(*singularityTaskData)
	if !ok {
		return errors.Errorf("Modification record %#v doesn't contain Singularity compatible data: was %T\n\t%#v", pair.ID(), data, pair)
	}
	reqID := data.requestID

	Log.Vomit.Printf("Operating on request %q", reqID)
	if r.changesReq(pair) {
		Log.Debug.Printf("Updating Request...")
		if err := r.Client.PostRequest(*pair.Post, reqID); err != nil {
			return err
		}
	} else {
		Log.Vomit.Printf("Request %q does not require changes", reqID)
	}

	if changesDep(pair) {
		Log.Debug.Printf("Deploying...")
		if err := r.Client.Deploy(*pair.Post, reqID); err != nil {
			return err
		}
	} else {
		Log.Vomit.Printf("Deploy on %q does not require change", reqID)
	}

	return nil
}

func (r deployer) changesReq(pair *sous.DeployablePair) bool {
	return pair.Prior.NumInstances != pair.Post.NumInstances
}

func changesDep(pair *sous.DeployablePair) bool {
	return pair.Post.Status == sous.DeployStatusFailed ||
		pair.Prior.Status == sous.DeployStatusFailed ||
		!(pair.Prior.SourceID.Equal(pair.Post.SourceID) &&
			pair.Prior.Resources.Equal(pair.Post.Resources) &&
			pair.Prior.Env.Equal(pair.Post.Env) &&
			pair.Prior.DeployConfig.Volumes.Equal(pair.Post.DeployConfig.Volumes))
}

func computeRequestID(d *sous.Deployable) (string, error) {
	return MakeRequestID(d.ID())
}

// MakeRequestID creates a Singularity request ID from a sous.DeploymentID.
func MakeRequestID(depID sous.DeploymentID) (string, error) {
	sn, err := depID.ManifestID.Source.ShortName()
	if err != nil {
		return "", err
	}
	sn = illegalDeployIDChars.ReplaceAllString(sn, "_")
	dd := illegalDeployIDChars.ReplaceAllString(depID.ManifestID.Source.Dir, "_")
	fl := illegalDeployIDChars.ReplaceAllString(depID.ManifestID.Flavor, "_")
	cl := illegalDeployIDChars.ReplaceAllString(depID.Cluster, "_")
	digest := depID.Digest()

	reqBase := fmt.Sprintf("%s-%s-%s-%s-%x", sn, dd, fl, cl, digest)

	if len(reqBase) > maxRequestIDLen {
		return reqBase[:(maxRequestIDLen)], nil
	}
	return reqBase, nil
}

func computeDeployID(d *sous.Deployable) string {
	var versionTrunc string
	uuidEntire := stripDeployID(uuid.NewV4().String())
	versionSansMeta := stripMetadata(d.Deployment.SourceID.Version.String())
	versionEntire := sanitizeDeployID(versionSansMeta)

	if len(versionEntire) > maxVersionLen {
		versionTrunc = versionEntire[0:maxVersionLen]
	} else {
		versionTrunc = versionEntire
	}

	depBase := strings.Join([]string{
		versionTrunc,
		uuidEntire,
	}, "_")

	if len(depBase) > maxDeployIDLen {
		return depBase[:(maxDeployIDLen)]
	}
	return depBase
}
