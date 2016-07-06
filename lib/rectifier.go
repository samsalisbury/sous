package sous

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/satori/go.uuid"
)

/*
The imagined use case here is like this:

intendedSet := getFromManifests()
existingSet := getFromSingularity()

dChans := intendedSet.Diff(existingSet)

Rectify(dChans)
*/

type (
	rectifier struct {
		Client RectificationClient
	}

	// RectificationClient abstracts the raw interactions with Singularity.  The
	// methods on this interface are tightly bound to the semantics of
	// Singularity itself - it's recommended to interact with the Sous Rectify
	// function or the rectification driver rather than with implentations of
	// this interface directly.
	// TODO: RectificationClient leaks Singularity concepts, make it so it doesn't.
	RectificationClient interface {
		// Deploy creates a new deploy on a particular requeust
		Deploy(cluster, depID, reqID, dockerImage string, r Resources, e Env, vols Volumes) error

		// PostRequest sends a request to a Singularity cluster to initiate
		PostRequest(cluster, reqID string, instanceCount int) error

		// Scale updates the instanceCount associated with a request
		Scale(cluster, reqID string, instanceCount int, message string) error

		// DeleteRequest instructs Singularity to delete a particular request
		DeleteRequest(cluster, reqID, message string) error

		//ImageName finds or guesses a docker image name for a Deployment
		ImageName(d *Deployment) (string, error)

		//ImageLabels finds the (sous) docker labels for a given image name
		ImageLabels(imageName string) (labels map[string]string, err error)
	}

	// DTOMap is shorthand for map[string]interface{}
	DTOMap map[string]interface{}

	// CreateError is returned when there's an error trying to create a deployment
	CreateError struct {
		Deployment *Deployment
		Err        error
	}

	// DeleteError is returned when there's an error while trying to delete a deployment
	DeleteError struct {
		Deployment *Deployment
		Err        error
	}

	// ChangeError describes an error that occurred while trying to change one deployment into another
	ChangeError struct {
		Deployments *DeploymentPair
		Err         error
	}

	// RectificationError is an interface that extends error with methods to get
	// the deployments the preceeded and were intended when the error occurred
	RectificationError interface {
		error
		ExistingDeployment() *Deployment
		IntendedDeployment() *Deployment
	}
)

func (e *CreateError) Error() string {
	return fmt.Sprintf("Couldn't create deployment %+v: %v", e.Deployment, e.Err)
}

// ExistingDeployment returns the deployment that was already existent in a change error
func (e *CreateError) ExistingDeployment() *Deployment {
	return nil
}

// IntendedDeployment returns the deployment that was intended in a ChangeError
func (e *CreateError) IntendedDeployment() *Deployment {
	return e.Deployment
}

func (e *DeleteError) Error() string {
	return fmt.Sprintf("Couldn't delete deployment %+v: %v", e.Deployment, e.Err)
}

// ExistingDeployment returns the deployment that was already existent in a change error
func (e *DeleteError) ExistingDeployment() *Deployment {
	return e.Deployment
}

// IntendedDeployment returns the deployment that was intended in a ChangeError
func (e *DeleteError) IntendedDeployment() *Deployment {
	return nil
}

func (e *ChangeError) Error() string {
	return fmt.Sprintf("Couldn't change from deployment %+v to deployment %+v: %v", e.Deployments.Prior, e.Deployments.Post, e.Err)
}

// ExistingDeployment returns the deployment that was already existent in a change error
func (e *ChangeError) ExistingDeployment() *Deployment {
	return e.Deployments.Prior
}

// IntendedDeployment returns the deployment that was intended in a ChangeError
func (e *ChangeError) IntendedDeployment() *Deployment {
	return e.Deployments.Post
}

// Rectify takes a DiffChans and issues the commands to the infrastructure to reconcile the differences
func Rectify(dcs DiffChans, rc RectificationClient) chan RectificationError {
	errs := make(chan RectificationError)
	rect := rectifier{rc}
	wg := &sync.WaitGroup{}
	wg.Add(3)
	go func() { rect.rectifyCreates(dcs.Created, errs); wg.Done() }()
	go func() { rect.rectifyDeletes(dcs.Deleted, errs); wg.Done() }()
	go func() { rect.rectifyModifys(dcs.Modified, errs); wg.Done() }()
	go func() { wg.Wait(); close(errs) }()

	return errs
}

func (r *rectifier) rectifyCreates(cc chan *Deployment, errs chan<- RectificationError) {
	for d := range cc {
		name, err := r.Client.ImageName(d)
		if err != nil {
			// log.Printf("% +v", d)
			errs <- &CreateError{Deployment: d, Err: err}
			continue
		}

		reqID := computeRequestID(d)
		err = r.Client.PostRequest(d.Cluster, reqID, d.NumInstances)
		if err != nil {
			// log.Printf("%T %#v", d, d)
			errs <- &CreateError{Deployment: d, Err: err}
			continue
		}

		err = r.Client.Deploy(d.Cluster, newDepID(), reqID, name, d.Resources, d.Env, d.DeployConfig.Volumes)
		if err != nil {
			// log.Printf("% +v", d)
			errs <- &CreateError{Deployment: d, Err: err}
			continue
		}
	}
}

func (r *rectifier) rectifyDeletes(dc chan *Deployment, errs chan<- RectificationError) {
	for d := range dc {
		err := r.Client.DeleteRequest(d.Cluster, computeRequestID(d), "deleting request for removed manifest")
		if err != nil {
			errs <- &DeleteError{Deployment: d, Err: err}
			continue
		}
	}
}

func (r *rectifier) rectifyModifys(
	mc chan *DeploymentPair, errs chan<- RectificationError) {
	for pair := range mc {
		Log.Debug.Printf("Rectifying modify: \n  %+ v \n    =>  \n  %+ v", pair.Prior, pair.Post)
		if r.changesReq(pair) {
			Log.Debug.Printf("Scaling...")
			err := r.Client.Scale(
				pair.Post.Cluster,
				computeRequestID(pair.Post),
				pair.Post.NumInstances,
				"rectified scaling")
			if err != nil {
				errs <- &ChangeError{Deployments: pair, Err: err}
				continue
			}
		}

		if changesDep(pair) {
			Log.Debug.Printf("Deploying...")
			name, err := r.Client.ImageName(pair.Post)
			if err != nil {
				errs <- &ChangeError{Deployments: pair, Err: err}
				continue
			}

			err = r.Client.Deploy(
				pair.Post.Cluster,
				newDepID(),
				computeRequestID(pair.Prior),
				name,
				pair.Post.Resources,
				pair.Post.Env,
				pair.Post.DeployConfig.Volumes,
			)
			if err != nil {
				errs <- &ChangeError{Deployments: pair, Err: err}
				continue
			}
		}
	}
}

func (r rectifier) changesReq(pair *DeploymentPair) bool {
	return pair.Prior.NumInstances != pair.Post.NumInstances
}

func changesDep(pair *DeploymentPair) bool {
	return !(pair.Prior.SourceVersion.Equal(pair.Post.SourceVersion) &&
		pair.Prior.Resources.Equal(pair.Post.Resources) &&
		pair.Prior.Env.Equal(pair.Post.Env) &&
		pair.Prior.DeployConfig.Volumes.Equal(pair.Post.DeployConfig.Volumes))
}

func computeRequestID(d *Deployment) string {
	if len(d.RequestID) > 0 {
		return d.RequestID
	}
	return idify(d.SourceVersion.CanonicalName().String())
}

var notInIDRE = regexp.MustCompile(`[-/:]`)

func idify(in string) string {
	return notInIDRE.ReplaceAllString(in, "")
}

func newDepID() string {
	return idify(uuid.NewV4().String())
}
