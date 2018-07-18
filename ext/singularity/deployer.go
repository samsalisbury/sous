package singularity

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/opentable/go-singularity"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
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

// DefaultMaxHTTPConcurrencyPerServer is the default maximum number of
// concurrent HTTP requests to send per Singularity server.
// To configure per deployer, see OptMaxHTTPConcurrencyPerServer.
const DefaultMaxHTTPConcurrencyPerServer = 10

/*
The imagined use case here is like this:

intendedSet := getFromManifests()
existingSet := getFromSingularity()

dChans := intendedSet.Diff(existingSet)
*/

type (
	deployer struct {
		Client        rectificationClient
		singFac       func(string) singClient
		ReqsPerServer int
		log           logging.LogSink
	}

	// rectificationClient abstracts the raw interactions with Singularity.
	rectificationClient interface {
		// Deploy creates a new deploy on a particular requeust
		Deploy(d sous.Deployable, reqID, depID string) error

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
func NewDeployer(c rectificationClient, ls logging.LogSink, options ...DeployerOption) sous.Deployer {
	d := &deployer{Client: c, log: ls, ReqsPerServer: DefaultMaxHTTPConcurrencyPerServer}
	for _, opt := range options {
		opt(d)
	}
	return d
}

// Rectify invokes actions to ensure that the real world matches pair.Post,
// given that it currently matches pair.Prior.
func (r *deployer) Rectify(pair *sous.DeployablePair) sous.DiffResolution {
	postID := ""
	version := ""
	var user sous.User
	if pair.Post != nil {
		postID = pair.Post.ID().String()
		version = pair.Post.DeploySpec().Version.String()
		user = pair.Post.Deployment.User
	}

	if pair.UUID == uuid.Nil {
		pair.UUID = uuid.NewV4()
	}

	switch k := pair.Kind(); k {
	default:
		panic(fmt.Sprintf("unrecognised kind %q", k))
	case sous.SameKind:
		resolution := pair.SameResolution()
		if pair.Post.Status == sous.DeployStatusFailed {
			resolution.Error = sous.WrapResolveError(&sous.FailedStatusError{})
		}

		messages.ReportLogFieldsMessage("SameKind", logging.InformationLevel, r.log, postID, version, resolution)
		return resolution
	case sous.AddedKind:
		messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Starting an AddedKind %s:%s by user %s", postID, version, user), logging.ExtraDebug1Level, r.log, pair)
		result := sous.DiffResolution{DeploymentID: pair.ID()}
		if err := r.RectifySingleCreate(pair); err != nil {
			result.Desc = "not created"
			switch t := err.(type) {
			default:
				result.Error = sous.WrapResolveError(&sous.CreateError{Deployment: pair.Post.Deployment.Clone(), Err: err})
			case *swaggering.ReqError:
				if t.Status == 400 {
					result.Error = sous.WrapResolveError(err)
				} else {
					result.Error = sous.WrapResolveError(&sous.CreateError{Deployment: pair.Post.Deployment.Clone(), Err: err})
				}
			}
		} else {
			result.Desc = sous.CreateDiff
		}
		messages.ReportLogFieldsMessage("Result of create", logging.InformationLevel, r.log, postID, version, result)
		reportDiffResolutionMessage("Result of create", result, logging.InformationLevel, r.log)
		return result
	case sous.RemovedKind:
		messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Starting an RemoveKind %s:%s by user %s", postID, version, user), logging.ExtraDebug1Level, r.log, pair)
		result := sous.DiffResolution{DeploymentID: pair.ID()}
		if err := r.RectifySingleDelete(pair); err != nil {
			result.Error = sous.WrapResolveError(&sous.DeleteError{Deployment: pair.Prior.Deployment.Clone(), Err: err})
			result.Desc = "not deleted"
		} else {
			result.Desc = sous.DeleteDiff
		}
		reportDiffResolutionMessage("Result of delete", result, logging.InformationLevel, r.log)
		messages.ReportLogFieldsMessage("Result of delete", logging.InformationLevel, r.log, postID, version, result)
		return result
	case sous.ModifiedKind:
		result := sous.DiffResolution{DeploymentID: pair.ID()}
		messages.ReportLogFieldsMessageToConsole(fmt.Sprintf("Starting a ModifiedKind %s:%s by user %s", postID, version, user), logging.ExtraDebug1Level, r.log, pair)
		if err := r.RectifySingleModification(pair); err != nil {
			dp := &sous.DeploymentPair{
				Prior: pair.Prior.Deployment.Clone(),
				Post:  pair.Post.Deployment.Clone(),
			}
			result.Error = sous.WrapResolveError(&sous.ChangeError{Deployments: dp, Err: err})
			result.Desc = "not updated"
		} else if pair.Prior.Status == sous.DeployStatusFailed || pair.Post.Status == sous.DeployStatusFailed {
			result.Desc = sous.ModifyDiff
			result.Error = sous.WrapResolveError(&sous.FailedStatusError{})
		} else {
			result.Desc = sous.ModifyDiff
		}
		reportDiffResolutionMessage("Result of modify", result, logging.InformationLevel, r.log)
		messages.ReportLogFieldsMessage("Result of modify", logging.InformationLevel, r.log, postID, version, result)
		return result
	}
}

func (r *deployer) SetSingularityFactory(fn func(string) singClient) {
	r.singFac = fn
}

func (r *deployer) buildSingClient(url string) singClient {
	if r.singFac == nil {
		return singularity.NewClient(url, r.log)
	}
	return r.singFac(url)
}

func rectifyRecover(d interface{}, f string, err *error, log logging.LogSink) {

	if r := recover(); r != nil {
		stack := string(debug.Stack())
		messages.ReportLogFieldsMessage("Panic", logging.WarningLevel, log, d, f, err, r, stack)
		*err = errors.Errorf("Panicked: %s; stack trace:\n%s", r, stack)
	}
}

func (r *deployer) RectifySingleCreate(d *sous.DeployablePair) (err error) {
	reportDeployerMessage("Rectifying creation", d, nil, nil, nil, logging.InformationLevel, r.log)
	defer rectifyRecover(d, "RectifySingleCreate", &err, r.log)
	if err != nil {
		return err
	}

	reqID := d.Post.Deployment.DeployConfig.SingularityRequestID
	if reqID == "" {
		reqID, err = computeRequestID(d.Post)
		if err != nil {
			return err
		}
	}

	if err = r.Client.PostRequest(*d.Post, reqID); err != nil {
		return err
	}
	depID := computeDeployIDFromUUID(d.Post, d.UUID)

	return r.Client.Deploy(*d.Post, reqID, depID)
}

func (r *deployer) RectifySingleDelete(d *sous.DeployablePair) (err error) {
	defer rectifyRecover(d, "RectifySingleDelete", &err, r.log)
	data, ok := d.ExecutorData.(*singularityTaskData)
	if !ok {
		return errors.Errorf("Delete record %#v doesn't contain Singularity compatible data: was %T\n\t%#v", d.ID(), data, d)
	}

	// TODO: Alert the owner of this request that there is no manifest for it;
	// they should either delete the request manually, or else add the manifest back.
	//LH note this function seems incomplete at the moment, could benefit some revision SS, JL, JC
	reportDeployerMessage("Rectify not deleting request", d, nil, data, nil, logging.WarningLevel, r.log)

	return nil
	// The following line deletes requests when it is not commented out.
	//return r.Client.DeleteRequest(d.Cluster.BaseURL, requestID, "deleting request for removed manifest")
}

func (r *deployer) RectifySingleModification(pair *sous.DeployablePair) (err error) {
	different, diffs := pair.Post.Deployment.Diff(pair.Prior.Deployment)
	if different {
		reportDeployerMessage("Rectifying modified diffs", pair, diffs, nil, nil, logging.InformationLevel, r.log)
	} else {
		reportDeployerMessage("Attempting to rectify empty diff", pair, diffs, nil, nil, logging.WarningLevel, r.log)
	}

	defer rectifyRecover(pair, "RectifySingleModification", &err, r.log)

	data, ok := pair.ExecutorData.(*singularityTaskData)
	if !ok {
		return errors.Errorf("Modification record %#v doesn't contain Singularity compatible data: was %T\n\t%#v", pair.ID(), data, pair)
	}
	currentReqID := data.requestID
	desiredReqID := pair.Post.Deployment.DeployConfig.SingularityRequestID

	if desiredReqID == "" {
		desiredReqID = currentReqID
	}

	if desiredReqID != currentReqID {
		// NOTE: This message is WarningLevel whilst the feature is new.
		// TODO SS: Turn this down to debug level once we're happy with it.
		m := fmt.Sprintf("Creating request %q to replace %q", desiredReqID, currentReqID)
		reportDeployerMessage(m, pair, diffs, data, nil, logging.WarningLevel, r.log)

		if err := r.Client.PostRequest(*pair.Post, desiredReqID); err != nil {
			return err
		}

		reportDeployerMessage("Deploying", pair, diffs, data, nil, logging.DebugLevel, r.log)
		depID := computeDeployIDFromUUID(pair.Post, pair.UUID)
		if err := r.Client.Deploy(*pair.Post, desiredReqID, depID); err != nil {
			return err
		}
		// TODO: Remove the old request.
		//m = fmt.Sprintf("Renamed to %q", desiredReqID)
		//return r.Client.DeleteRequest(pair.Post.Deployment.Cluster.BaseURL, currentReqID, m)
		return nil
	}

	reportDeployerMessage("Operating on request", pair, diffs, data, nil, logging.ExtraDebug1Level, r.log)
	if changesReq(pair) {
		reportDeployerMessage("Updating request", pair, diffs, data, nil, logging.DebugLevel, r.log)
		if err := r.Client.PostRequest(*pair.Post, desiredReqID); err != nil {
			return err
		}
	} else {
		reportDeployerMessage("No change to Singularity request required", pair, diffs, data, nil, logging.DebugLevel, r.log)
	}

	if changesDep(pair) {
		reportDeployerMessage("Deploying", pair, diffs, data, nil, logging.DebugLevel, r.log)
		depID := computeDeployIDFromUUID(pair.Post, pair.UUID)
		if err := r.Client.Deploy(*pair.Post, desiredReqID, depID); err != nil {
			return err
		}
	} else {
		reportDeployerMessage("No change to Singularity deployment required", pair, diffs, data, nil, logging.DebugLevel, r.log)
	}

	return nil
}

// XXX for logging and other UI purposes, the best thing would be if the
// DeployablePair had a "diff" method that returned a (cached) list of
// differences, which these two functions could filter for req/dep triggering
// changes. Then, rather than simply computing the conditional, the deployer
// could report ("deploy required because of %v", diffs)

func changesReq(pair *sous.DeployablePair) bool {
	return (pair.Prior.Kind == sous.ManifestKindScheduled && pair.Prior.Schedule != pair.Post.Schedule) ||
		pair.Prior.Kind != pair.Post.Kind ||
		pair.Prior.NumInstances != pair.Post.NumInstances ||
		!pair.Prior.Owners.Equal(pair.Post.Owners)
}

func changesDep(pair *sous.DeployablePair) bool {
	return pair.Post.Status == sous.DeployStatusFailed ||
		pair.Prior.Status == sous.DeployStatusFailed ||
		!(pair.Prior.SourceID.Equal(pair.Post.SourceID) &&
			pair.Prior.Resources.Equal(pair.Post.Resources) &&
			pair.Prior.Env.Equal(pair.Post.Env) &&
			pair.Prior.DeployConfig.Volumes.Equal(pair.Post.DeployConfig.Volumes) &&
			pair.Prior.Startup.Equal(pair.Post.Startup))
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

// MakeRequestURL creates a singularity request url
func MakeRequestURL(baseURL string, requestID string) (string, error) {
	if len(baseURL) == 0 {
		return "", errors.Errorf("baseURL can not be empty : %s", baseURL)
	}
	if len(requestID) == 0 {
		return "", errors.Errorf("requestID can not be empty : %s", requestID)
	}
	return fmt.Sprintf("%s/request/%s", baseURL, requestID), nil
}

func computeDeployID(d *sous.Deployable) string {
	return computeDeployIDFromUUID(d, uuid.NewV4())
}

func computeDeployIDFromUUID(d *sous.Deployable, uid uuid.UUID) string {
	var versionTrunc string
	uuidEntire := stripDeployID(uid.String())
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
