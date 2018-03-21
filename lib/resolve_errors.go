package sous

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/opentable/sous/util/logging"
	"github.com/pkg/errors"
)

type (
	// ResolveErrors collect all the errors for a resolve action into a single
	// error to be handled elsewhere
	ResolveErrors struct {
		Causes []ErrorWrapper
	}

	// ErrorWrapper wraps an error so that it can be marshalled and unmarshalled
	// to JSON
	ErrorWrapper struct {
		MarshallableError
		error
	}

	// MarshallableError captures parts of an error that can be serialized
	// successfully
	MarshallableError struct {
		Type, String string
	}
	// MissingImageNameError reports that we couldn't get names for one or
	// more source IDs.
	MissingImageNameError struct {
		Cause error
	}

	// A FailedStatusError reports that the the deploy has reported as failed on
	// singularity
	FailedStatusError struct{} // XXX maybe handy to have the root Singularity non-SUCCEEDED status?

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

// WrapResolveError wraps an error inside an *ErrorWrapper for marshaling.
func WrapResolveError(err error) *ErrorWrapper {
	if ew, is := err.(*ErrorWrapper); is {
		return ew
	}
	return &ErrorWrapper{error: err}
}

// MarshalJSON implements json.Marshaller on ErrorWrapper.
// It makes sure that the embedded MarshallableError is populated and then
// marshals that. The upshot is that errors can be successfully marshalled into
// JSON for review by the client.
func (ew ErrorWrapper) MarshalJSON() ([]byte, error) {
	ew.MarshallableError = buildMarshableError(ew.error)
	return json.Marshal(ew.MarshallableError)
}

func (ew ErrorWrapper) Error() string {
	if ew.error != nil {
		return ew.error.Error()
	}
	return ew.String
}

func buildMarshableError(err error) MarshallableError {
	ew := MarshallableError{}
	ew.Type = fmt.Sprintf("%T", err)
	if err != nil {
		ew.String = fmt.Sprintf("%s", err.Error())
	} else {
		ew.String = "Failed to marshal error, it was NULL"
	}
	return ew
}

func (re *ResolveErrors) Error() string {
	s := []string{"Errors during resolve:"}
	for _, e := range re.Causes {
		s = append(s, e.Error())
	}
	return strings.Join(s, "\n  ")
}

// AnyTransientResolveErrors returns true for transient resolve errors
// (see code for details).
// The intention is to filter Resolver.Begin(...).Wait() results for loops,
// so that transient errors can be retried.
func AnyTransientResolveErrors(err error) bool {
	switch te := errors.Cause(err).(type) {
	default:
		return false
	case *ResolveErrors:
		for _, e := range te.Causes {
			if IsTransientResolveError(e) {
				return true
			}
		}
		return false
	}
}

// IsTransientResolveError returns true for resolve errors which might resolve on
// their own. All other errors, it returns false
func IsTransientResolveError(err error) bool {
	switch terr := errors.Cause(err).(type) {
	default:
		// unnamed errors are by definition not resolve errors
		return false
	case *ErrorWrapper:
		// ErrorWrappers carry string data about an error across an HTTP
		// transaction.  We basically need to check it's Type field.

		// First, if it has a non-nil error, it hasn't been serialized yet.
		// Use the live error.
		if terr.error != nil {
			return IsTransientResolveError(terr.error)
		}

		// If not, then we need to rely on the strings.
		logging.Log.Vomit.Printf("Checking err string type: %s", terr.Type)
		switch terr.Type {
		default:
			return false
		case "*sous.ChangeError":
			return true
		case "*sous.CreateError":
			return true
		}

	case *FailedStatusError:
		// Anything but SUCCEEDED on Singularity is a failure for this deploy.
		// There's no expectation that it will self correct. In the future, we
		// should do a automatic rollback.
		return false
	case *UnacceptableAdvisory:
		// UnacceptableAdvisory is excluded, since this requires operator
		// intervention: either the image needs to be rebuilt clean, or the cluster
		// reconfigured to accept the advisory.
		return false
	case *MissingImageNameError:
		// MissingImageNameError isn't transient: it requires that an appropriate
		// image be built with the desired name and the server needs to be able to
		// at least guess the image's name.
		return false
	case *ChangeError:
		// ChangeError is typically returned when Singularity returns an error (which we don't yet
		// distinguish - empirically, this most often means that a particular Request
		// is in the midst of deploying and not accepting new Deploys yet.)
		return true
	case *CreateError:
		// CreateErrors are returned when Singularity returns errors when we try to
		// create a request or deploy. This might be the result of a conflicting
		// Request name, in which case it's likely that the next attempt to resolve
		// will be a Modify instead.
		return true
	case *DeleteError:
		// XXX While "deletes" are no-ops, there's no chance that a DeleteError is going to "self correct"
		//		return true
		return false // XXX
	}
}

func (e *MissingImageNameError) Error() string {
	return fmt.Sprintf("Image name unknown to Sous for source IDs: %s", e.Cause.Error())
}

func (e *UnacceptableAdvisory) Error() string {
	return fmt.Sprintf("Advisory unacceptable on image: %s for %v", e.Quality.Name, e.SourceID)
}

func (e *FailedStatusError) Error() string {
	return "Deploy failed on Singularity."
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
