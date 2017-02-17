package singularity

import (
	"fmt"
	"log"
	"runtime/debug"
	"strings"

	"github.com/opentable/go-singularity"
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

// Singularity DeployID must be <50
const maxDeployIDLen = 49

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
		Client                   rectificationClient
		SingularityClientFactory func(*sous.Cluster) *singularity.Client
	}

	// rectificationClient abstracts the raw interactions with Singularity.
	rectificationClient interface {
		// Deploy creates a new deploy on a particular requeust
		Deploy(cluster, depID, reqID, dockerImage string, r sous.Resources, e sous.Env, vols sous.Volumes) error

		// PostRequest sends a request to a Singularity cluster to initiate
		PostRequest(cluster, reqID string, instanceCount int, kind sous.ManifestKind, owners sous.OwnerSet) error

		// DeleteRequest instructs Singularity to delete a particular request
		DeleteRequest(cluster, reqID, message string) error
	}

	// DTOMap is shorthand for map[string]interface{}
	dtoMap map[string]interface{}
)

// NewDeployer creates a new Singularity-based sous.Deployer.
func NewDeployer(c rectificationClient, singularityClientFactory func(*sous.Cluster) *singularity.Client) sous.Deployer {
	return &deployer{Client: c, SingularityClientFactory: singularityClientFactory}
}

func (r *deployer) RunningDeployments(reg sous.Registry, clusters sous.Clusters) (sous.DeployStates, error) {
	if len(clusters) != 1 {
		return sous.NewDeployStates(), fmt.Errorf("RunningDeployments needs exactly one cluster")
	}
	var cluster sous.Cluster
	for _, c := range clusters {
		if c == nil {
			return sous.NewDeployStates(), fmt.Errorf("nil cluster")
		}
		cluster = *c
	}

	newDeployer := &Deployer{
		Registry: reg,
		Client:   r.SingularityClientFactory(&cluster),
		Cluster:  cluster,
	}
	ds, err := newDeployer.RunningDeployments()
	if err != nil {
		return sous.NewDeployStates(), err
	}
	return *ds, nil
}

// RectifyCreates rectifies newly created Singularity requests.
func (r *deployer) RectifyCreates(cc <-chan *sous.Deployable, errs chan<- sous.DiffResolution) {
	for d := range cc {
		result := sous.DiffResolution{DeployID: d.ID()}
		if err := r.RectifySingleCreate(d); err != nil {
			result.Error = sous.WrapResolveError(&sous.CreateError{Deployment: d.Deployment, Err: err})
			result.Desc = "not created"
		} else {
			result.Desc = "created"
		}
		errs <- result
	}
}

// ImageName returns the Docker image name for a Deployable.
func (r *deployer) ImageName(d *sous.Deployable) (string, error) {
	if d.BuildArtifact == nil {
		return "", &sous.MissingImageNameError{Cause: fmt.Errorf("Missing BuildArtifact on Deployable")}
	}
	return d.BuildArtifact.Name, nil
}

func rectifyRecover(d interface{}, f string, err *error) {
	if r := recover(); r != nil {
		sous.Log.Warn.Printf("Panic in %s with %# v", f, d)
		sous.Log.Warn.Printf("  %v", r)
		sous.Log.Warn.Print(string(debug.Stack()))
		*err = errors.Errorf("Panicked")
	}
}

// RectifySingleCreate rectifies a single new Singularity Request creation.
func (r *deployer) RectifySingleCreate(d *sous.Deployable) (err error) {
	Log.Debug.Printf("Rectifying creation %q:  \n %# v", d.ID(), d.Deployment)
	defer rectifyRecover(d, "RectifySingleCreate", &err)
	name, err := r.ImageName(d)
	if err != nil {
		return err
	}
	reqID := computeRequestID(d)
	if err = r.Client.PostRequest(d.Cluster.BaseURL, reqID, d.NumInstances, d.Kind, d.Owners); err != nil {
		return err
	}
	return r.Client.Deploy(
		d.Cluster.BaseURL, computeDeployID(d), reqID, name, d.Resources,
		d.Env, d.DeployConfig.Volumes)
}

// RectifyDeletes rectifies all Singularity Request deletions.
func (r *deployer) RectifyDeletes(dc <-chan *sous.Deployable, errs chan<- sous.DiffResolution) {
	for d := range dc {
		result := sous.DiffResolution{DeployID: d.ID()}
		if err := r.RectifySingleDelete(d); err != nil {
			result.Error = sous.WrapResolveError(&sous.DeleteError{Deployment: d.Deployment, Err: err})
			result.Desc = "not deleted"
		} else {
			result.Desc = "deleted"
		}
		errs <- result
	}
}

// RectifySingleDelete rectifies a single Singularity Request deletion.
func (r *deployer) RectifySingleDelete(d *sous.Deployable) (err error) {
	defer rectifyRecover(d, "RectifySingleDelete", &err)
	requestID := computeRequestID(d)
	// TODO: Alert the owner of this request that there is no manifest for it;
	// they should either delete the request manually, or else add the manifest back.
	sous.Log.Warn.Printf("NOT DELETING REQUEST %q (FOR: %q)", requestID, d.ID())
	return nil
	// The following line deletes requests when it is not commented out.
	//return r.Client.DeleteRequest(d.Cluster.BaseURL, requestID, "deleting request for removed manifest")
}

// RectifyModifies rectifies all modifications to existing deployments and new
// deployments for a request.
func (r *deployer) RectifyModifies(
	mc <-chan *sous.DeployablePair, errs chan<- sous.DiffResolution) {
	for pair := range mc {
		result := sous.DiffResolution{DeployID: pair.ID()}
		if err := r.RectifySingleModification(pair); err != nil {
			dp := &sous.DeploymentPair{
				Prior: pair.Prior.Deployment,
				Post:  pair.Post.Deployment,
			}
			log.Printf("%#v", err)
			result.Error = sous.WrapResolveError(&sous.ChangeError{Deployments: dp, Err: err})
			result.Desc = "not updated"
		} else {
			result.Desc = "updated"
		}
		errs <- result
	}
}

// RectifySingleModification rectifies a single request modification or creates
// a new deployment.
func (r *deployer) RectifySingleModification(pair *sous.DeployablePair) (err error) {
	Log.Debug.Printf("Rectifying modified %q: \n  %# v \n    =>  \n  %# v", pair.ID(), pair.Prior.Deployment, pair.Post.Deployment)
	defer rectifyRecover(pair, "RectifySingleModification", &err)
	if r.changesReq(pair) {
		Log.Debug.Printf("Updating Request...")
		if err := r.Client.PostRequest(
			pair.Post.Cluster.BaseURL,
			computeRequestID(pair.Post),
			pair.Post.NumInstances,
			pair.Post.Kind,
			pair.Post.Owners,
		); err != nil {
			return err
		}
	}

	if changesDep(pair) {
		Log.Debug.Printf("Deploying...")
		name, err := r.ImageName(pair.Post)
		if err != nil {
			return err
		}

		if err := r.Client.Deploy(
			pair.Post.Cluster.BaseURL,
			computeDeployID(pair.Post),
			computeRequestID(pair.Prior),
			name,
			pair.Post.Resources,
			pair.Post.Env,
			pair.Post.DeployConfig.Volumes,
		); err != nil {
			return err
		}
	}
	return nil
}

func (r deployer) changesReq(pair *sous.DeployablePair) bool {
	return pair.Prior.NumInstances != pair.Post.NumInstances
}

func changesDep(pair *sous.DeployablePair) bool {
	return !(pair.Prior.SourceID.Equal(pair.Post.SourceID) &&
		pair.Prior.Resources.Equal(pair.Post.Resources) &&
		pair.Prior.Env.Equal(pair.Post.Env) &&
		pair.Prior.DeployConfig.Volumes.Equal(pair.Post.DeployConfig.Volumes))
}

func computeRequestID(d *sous.Deployable) string {
	if len(d.RequestID) > 0 {
		return d.RequestID
	}
	return MakeRequestID(d.ID())
}

func computeDeployID(d *sous.Deployable) string {
	var uuidTrunc, versionTrunc string
	uuidEntire := StripDeployID(uuid.NewV4().String())
	versionSansMeta := stripMetadata(d.Deployment.SourceID.Version.String())
	versionEntire := SanitizeDeployID(versionSansMeta)

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

// MakeRequestID creats a Singularity request ID from a sous.DeployID.
func MakeRequestID(mid sous.DeployID) string {
	sl := strings.Replace(mid.ManifestID.Source.String(), "/", ">", -1)
	return fmt.Sprintf("%s:%s:%s", sl, mid.ManifestID.Flavor, mid.Cluster)
}

// ParseRequestID parses a DeployID from a Singularity request ID created by
// Sous.
func ParseRequestID(id string) (sous.DeployID, error) {
	parts := strings.Split(id, ":")
	if len(parts) != 3 {
		return sous.DeployID{}, fmt.Errorf("request ID %q should contain exactly 2 colons", id)
	}
	if len(parts[0]) == 0 {
		return sous.DeployID{}, fmt.Errorf("request ID %q has an empty SourceLocation", id)
	}
	if len(parts[2]) == 0 {
		return sous.DeployID{}, fmt.Errorf("request ID %q has an empty Cluster name", id)
	}
	parts[0] = strings.Replace(parts[0], ">", "/", -1)
	slParts := strings.Split(parts[0], ",")
	if len(slParts) == 1 {
		slParts = append(slParts, "")
	}

	return sous.DeployID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: slParts[0],
				Dir:  slParts[1],
			},
			Flavor: parts[1],
		},
		Cluster: parts[2],
	}, nil
}
