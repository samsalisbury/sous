package singularity

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/opentable/go-singularity"
	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

/*
The imagined use case here is like this:

intendedSet := getFromManifests()
existingSet := getFromSingularity()

dChans := intendedSet.Diff(existingSet)
*/

type (
	deployer struct {
		Client   rectificationClient
		Registry sous.Registry
		singFac  func(string) *singularity.Client
	}

	// rectificationClient abstracts the raw interactions with Singularity.
	rectificationClient interface {
		// Deploy creates a new deploy on a particular requeust
		Deploy(cluster, depID, reqID, dockerImage string, r sous.Resources, e sous.Env, vols sous.Volumes) error

		// PostRequest sends a request to a Singularity cluster to initiate
		PostRequest(cluster, reqID string, instanceCount int) error

		// Scale updates the instanceCount associated with a request
		Scale(cluster, reqID string, instanceCount int, message string) error

		// DeleteRequest instructs Singularity to delete a particular request
		DeleteRequest(cluster, reqID, message string) error
	}

	// DTOMap is shorthand for map[string]interface{}
	dtoMap map[string]interface{}
)

// NewDeployer creates a new Singularity-based sous.Deployer.
func NewDeployer(r sous.Registry, c rectificationClient) sous.Deployer {
	return &deployer{Client: c, Registry: r}
}

func (r *deployer) RectifyCreates(cc <-chan *sous.Deployment, errs chan<- sous.RectificationError) {
	for d := range cc {
		if err := r.RectifySingleCreate(d); err != nil {
			errs <- &sous.CreateError{Deployment: d, Err: err}
		}
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

func (r *deployer) ImageName(d *sous.Deployment) (string, error) {
	a, err := r.Registry.GetArtifact(d.SourceID)
	if err != nil {
		return "", err
	}
	return a.Name, err
}

func (r *deployer) RectifySingleCreate(d *sous.Deployment) (err error) {
	Log.Debug.Printf("Rectifing create:  \n %# v", d)
	defer func(fd *sous.Deployment) {
		if r := recover(); r != nil {
			sous.Log.Warn.Printf("Panic in RectifySingleCreate with %# v", fd)
			sous.Log.Warn.Printf("  %v", r)
			sous.Log.Warn.Print(string(debug.Stack()))
			err = errors.Errorf("Panicked")
		}
	}(d)
	name, err := r.ImageName(d)
	if err != nil {
		return err
	}
	reqID := computeRequestID(d)
	if err = r.Client.PostRequest(d.Cluster.BaseURL, reqID, d.NumInstances); err != nil {
		return err
	}
	return r.Client.Deploy(
		d.Cluster.BaseURL, newDepID(), reqID, name, d.Resources,
		d.Env, d.DeployConfig.Volumes)
}

func (r *deployer) RectifyDeletes(dc <-chan *sous.Deployment, errs chan<- sous.RectificationError) {
	for d := range dc {
		if err := r.RectifySingleDelete(d); err != nil {
			errs <- &sous.DeleteError{Deployment: d, Err: err}
		}
	}
}

func (r *deployer) RectifySingleDelete(d *sous.Deployment) (err error) {
	defer func(fd *sous.Deployment) {
		if r := recover(); r != nil {
			sous.Log.Warn.Printf("Panic in RectifySingleDelete with %# v", fd)
			sous.Log.Warn.Printf("  %v", r)
			sous.Log.Warn.Print(string(debug.Stack()))
			err = errors.Errorf("Panicked")
		}
	}(d)
	return r.Client.DeleteRequest(d.Cluster.BaseURL, computeRequestID(d), "deleting request for removed manifest")
}

func (r *deployer) RectifyModifies(
	mc <-chan *sous.DeploymentPair, errs chan<- sous.RectificationError) {
	for pair := range mc {
		if err := r.RectifySingleModification(pair); err != nil {
			errs <- &sous.ChangeError{Deployments: pair, Err: err}
		}
	}
}

func (r *deployer) RectifySingleModification(pair *sous.DeploymentPair) (err error) {
	Log.Debug.Printf("Rectifying modify: \n  %# v \n    =>  \n  %# v", pair.Prior, pair.Post)
	defer func(fp *sous.DeploymentPair) {
		if r := recover(); r != nil {
			sous.Log.Warn.Printf("Panic in RectifySingleModification with %# v", fp)
			sous.Log.Warn.Printf("  %v", r)
			sous.Log.Warn.Print(string(debug.Stack()))
			err = errors.Errorf("Panicked")
		}
	}(pair)
	if r.changesReq(pair) {
		Log.Debug.Printf("Scaling...")
		if err := r.Client.Scale(
			pair.Post.Cluster.BaseURL,
			computeRequestID(pair.Post),
			pair.Post.NumInstances,
			"rectified scaling"); err != nil {
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
			newDepID(),
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

func (r deployer) changesReq(pair *sous.DeploymentPair) bool {
	return pair.Prior.NumInstances != pair.Post.NumInstances
}

func changesDep(pair *sous.DeploymentPair) bool {
	return !(pair.Prior.SourceID.Equal(pair.Post.SourceID) &&
		pair.Prior.Resources.Equal(pair.Post.Resources) &&
		pair.Prior.Env.Equal(pair.Post.Env) &&
		pair.Prior.DeployConfig.Volumes.Equal(pair.Post.DeployConfig.Volumes))
}

func computeRequestID(d *sous.Deployment) string {
	if len(d.RequestID) > 0 {
		return d.RequestID
	}
	return MakeRequestID(d.ID())
}

// MakeRequestID creats a Singularity request ID from a sous.DeployID.
func MakeRequestID(mid sous.DeployID) string {
	sl := strings.Replace(mid.ManifestID.Source.String(), "/", ">", -1)
	return fmt.Sprintf("%s:%s:%s", sl, mid.ManifestID.Flavor, mid.Cluster)
}

func newDepID() string {
	return MakeDeployID(uuid.NewV4().String())
}
