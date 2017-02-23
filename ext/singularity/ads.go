package singularity

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/coaxer"
)

// c is a temporary global, it will be moved somewhere more sensible soon.
var c = coaxer.NewCoaxer(func(c *coaxer.Coaxer) {
	messages := make(chan string)
	go func() {
		for m := range messages {
			log.Println(m)
		}
	}()
	c.DebugFunc = func(desc string) {
		messages <- desc
	}
	c.Backoff = time.Second
})

// DeployReader encapsulates the methods required to read Singularity
// requests and deployments.
type DeployReader interface {
	GetRequests() (dtos.SingularityRequestParentList, error)
	GetRequest(requestID string) (*dtos.SingularityRequestParent, error)
	GetDeploy(requestID, deployID string) (*dtos.SingularityDeployHistory, error)
	GetDeploys(requestID string, count, page int32) (dtos.SingularityDeployHistoryList, error)
}

// Deployer implements sous.Deployer for a single sous Cluster running on
// Singularity.
type Deployer struct {
	Registry      sous.Registry
	ClientFactory func(*sous.Cluster) DeployReader
	Clusters      sous.Clusters
}

// adsBuild represents the building of a single sous.DeployStates from a
// single Singularity-hosted cluster.
type adsBuild struct {
	Context       context.Context
	ClientFactory func(*sous.Cluster) DeployReader
	Clusters      sous.Clusters
	Registry      sous.Registry
	ErrorCallback func(error)
}

// newRequestContext initialises a requestContext and begins making HTTP
// requests to get the request (via coaxer). We can access the results of
// this via the returned requestContext's promise field.
func (ab *adsBuild) newRequestContext(requestID string, cluster *sous.Cluster) *requestContext {
	return newRequestContext(
		ab.Context, requestID, ab.ClientFactory(cluster), *cluster, ab.Registry,
	)
}

func newADSBuild(ctx context.Context, client func(*sous.Cluster) DeployReader, reg sous.Registry, clusters sous.Clusters) *adsBuild {
	return &adsBuild{
		ClientFactory: client,
		Registry:      reg,
		Clusters:      clusters,
		ErrorCallback: func(err error) { log.Println(err) },
		Context:       ctx,
	}
}

// RunningDeployments uses a new adsBuild to construct sous deploy states.
func (d *Deployer) RunningDeployments() (sous.DeployStates, error) {
	return newADSBuild(context.TODO(), d.ClientFactory, d.Registry, d.Clusters).DeployStates()
}

// DeployStates returns all deploy states.
func (ab *adsBuild) DeployStates() (sous.DeployStates, error) {

	log.Printf("Getting all requests...")

	promises := make(map[string]coaxer.Promise, len(ab.Clusters))

	var requests []*dtos.SingularityRequestParent

	// Grab the list of all requests from all clusters.
	for clusterName, cluster := range ab.Clusters {
		cluster := cluster
		// TODO: Make sous.Clusters a slice to avoid this double-entry record keeping.
		cluster.Name = clusterName
		promises[cluster.Name] = c.Coax(context.TODO(), func() (interface{}, error) {
			if ab.ClientFactory == nil {
				panic("CF")
			}
			if ab.Clusters == nil {
				panic("CLUSTERS")
			}
			if cluster == nil {
				panic("CLUSTER")
			}
			client := ab.ClientFactory(ab.Clusters[cluster.Name])
			return maybeRetryable(client.GetRequests())
		}, "get requests from cluster %q", cluster.Name)
	}

	for cluster, promise := range promises {
		if err := promise.Err(); err != nil {
			log.Printf("Fatal: unable to get requests for cluster %q", cluster)
			return sous.NewDeployStates(), err
		}
		log.Printf("Success: got all requests from cluster %q", cluster)
		requests = append(requests, promise.Value().(dtos.SingularityRequestParentList)...)
	}

	log.Printf("Got: %d requests", len(requests))

	deployStates := sous.NewDeployStates()
	var wg sync.WaitGroup
	errChan := make(chan error)

	// Start gathering all requests concurrently.
gather:
	for _, request := range requests {
		request := request
		select {
		case <-ab.Context.Done():
			log.Printf("Context ended before all deployments gathered.")
			break gather
		default:
		}

		requestID := request.Request.Id

		log.Printf("Gathering data for request %q in background.", requestID)
		deployID, err := ParseRequestID(requestID)
		if err != nil {
			// TODO: Maybe log this?
			continue
		}
		oneOfMyDeploys := false
		for clusterName := range ab.Clusters {
			if deployID.Cluster == clusterName {
				oneOfMyDeploys = true
				break
			}
		}
		if !oneOfMyDeploys {
			// TODO: Maybe log this?
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			dsb, err := ab.newDeployStateBuilder(deployID.Cluster, request)
			if err != nil {
				ab.ErrorCallback(err)
				errChan <- err
				return
			}
			ds, err := dsb.DeployState()
			if err != nil {
				ab.ErrorCallback(err)
				errChan <- err
				return
			}
			deployStates.Add(ds)
		}()
	}

	go func() {
		wg.Wait()
		close(errChan)
	}()

	// Wait for either error or channel close.
	if err := <-errChan; err != nil {
		return sous.NewDeployStates(), err
	}

	return deployStates, nil
}

func (ab *adsBuild) Errorf(format string, a ...interface{}) error {
	//prefix := fmt.Sprintf("reading from cluster %q", ab.Cluster.Name)
	prefix := ""
	message := fmt.Sprintf(format, a...)
	return fmt.Errorf("%s: %s", prefix, message)
}

func (ab *adsBuild) newDeployStateBuilder(clusterName string, rp *dtos.SingularityRequestParent) (*DeployStateBuilder, error) {
	cluster := ab.Clusters[clusterName]
	requestID := rp.Request.Id
	rc := ab.newRequestContext(requestID, cluster)
	return newDeployStateBuilder(rc)
}

type temporary struct {
	error
}

func (t temporary) Temporary() bool {
	return true
}

func maybeRetryable(a interface{}, err error) (interface{}, error) {
	if err == nil {
		return a, nil
	}
	log.Printf("Maybe retryable %T? %q", err, err)
	return a, temporary{err}
}
