package sous

import (
	"fmt"
	"strings"
)

type (
	// ResolveErrors collect all the errors for a resolve action into a single
	// error to be handled elsewhere
	ResolveErrors struct {
		Causes []error
	}

	// MissingImageNameError reports that we couldn't get names for one or
	// more source IDs.
	MissingImageNameError struct {
		Cause error
	}

	// An UnacceptableAdvisory reports that there is an advisory on an image
	// which hasn't been whitelisted on the target cluster
	UnacceptableAdvisory struct {
		Quality
		*SourceID
	}

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

func (re *ResolveErrors) Error() string {
	s := []string{"Errors during resolve:"}
	for _, e := range re.Causes {
		s = append(s, e.Error())
	}
	return strings.Join(s, "\n  ")
}

func (e *MissingImageNameError) Error() string {
	return fmt.Sprintf("Image name unknown to Sous for source IDs: %s", e.Cause.Error())
}

func (e *UnacceptableAdvisory) Error() string {
	return fmt.Sprintf("Advisory unacceptable on image: %s for %v", e.Quality.Name, e.SourceID)
}

func (e *CreateError) Error() string {
	return fmt.Sprintf("Couldn't create deployment\n  %+v: %v", e.Deployment, e.Err)
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
	return fmt.Sprintf("%v: Couldn't delete deployment %+v", e.Err, e.Deployment)
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
	return fmt.Sprintf("%v: Couldn't change from deployment\n  %+v\n\n  to deployment\n\n  %+v", e.Err, e.Deployments.Prior, e.Deployments.Post)
}

// ExistingDeployment returns the deployment that was already existent in a change error
func (e *ChangeError) ExistingDeployment() *Deployment {
	return e.Deployments.Prior
}

// IntendedDeployment returns the deployment that was intended in a ChangeError
func (e *ChangeError) IntendedDeployment() *Deployment {
	return e.Deployments.Post
}
