package sous

import (
	"fmt"
	"log"
	"sync"

	"github.com/opentable/singularity"
	"github.com/opentable/singularity/dtos"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
)

const ReqsPerServer = 10

type singReq struct {
	sourceUrl string
	req       *dtos.SingularityRequestParent
}

func GetRunningDeploymentSet(singUrls []string) (deps Deployments, err error) {
	// XXX: the biggest issue I'm currently aware of here is how the various clients are managed.
	//      Be aware: a client is created for every image data request, even though they'll all be
	//      directed to the same server - fixing this would require changes inside the docker/distribution code.
	//      Maybe worse: if there's a problem, all outstanding HTTP requests will be allowed to complete, even though
	//      Sous may not be available to receive the response.
	//      The biggest concrete problem I can imagine with that at the moment is if Sous were to run in
	//      tight loop and fail, possibly exhausting connections at the docker registry.
	errCh := make(chan error)
	deps = make(Deployments, 0)

	reqCh := make(chan singReq, len(singUrls)*ReqsPerServer)
	depCh := make(chan Deployment, ReqsPerServer)
	defer close(depCh)

	regClient := docker_registry.NewClient()
	defer regClient.Cancel()

	var singWait, depWait sync.WaitGroup

	singWait.Add(len(singUrls))
	for _, url := range singUrls {
		go singPipeline(url, singWait, reqCh, errCh)
	}

	depWait.Add(1)
	go depPipeline(depWait, regClient, reqCh, depCh, errCh)

	go func() {
		catchAndSend("closing up", errCh)
		singWait.Wait()
		close(reqCh)

		depWait.Wait()
		close(errCh)
	}()

	for {
		select {
		case dep := <-depCh:
			deps = append(deps, dep)
			depWait.Done()
		case err = <-errCh:
			return
		}
	}
}

func catchAll(from string) {
	if err := recover(); err != nil {
		log.Printf("Recovering from %s where we received %v", from, err)
	}
}

func catchAndSend(from string, errs chan error) {
	defer catchAll(from)
	if err := recover(); err != nil {
		switch err := err.(type) {
		default:
			if err != nil {
				errs <- fmt.Errorf("Panicked with not-error: %v", err)
			}
		case error:
			errs <- err
		}
	}
}

func singPipeline(url string, wg sync.WaitGroup, reqs chan singReq, errs chan error) {
	defer wg.Done()
	defer catchAndSend(fmt.Sprintf("get requests: %s", url), errs)
	rs, err := getRequestsFromSingularity(url)
	if err != nil {
		errs <- err
		return
	}
	for _, r := range rs {
		reqs <- r
	}
}

func depPipeline(wait sync.WaitGroup, cl *docker_registry.Client, reqCh chan singReq, depCh chan Deployment, errCh chan error) {
	defer catchAndSend("dependency building", errCh)
	for req := range reqCh {
		wait.Add(1)
		go func() {
			defer catchAndSend(fmt.Sprintf("dep from req %s", req.sourceUrl), errCh)
			dep, err := deploymentFromRequest(cl, req)
			if err != nil {
				errCh <- err
			}

			depCh <- dep
		}()
	}
	wait.Done()
}

func getRequestsFromSingularity(singUrl string) ([]singReq, error) {
	client := singularity.NewClient(singUrl)
	singRequests, err := client.GetRequests()
	if err != nil {
		return nil, err
	}

	reqs := make([]singReq, 0, len(singRequests))
	for _, sr := range singRequests {
		reqs = append(reqs, singReq{singUrl, sr})
	}

	return reqs, nil
}

func deploymentFromRequest(cl *docker_registry.Client, sr singReq) (Deployment, error) {
	cluster := sr.sourceUrl
	req := sr.req

	singDep := req.ActiveDeploy
	env := singDep.Env
	singRez := singDep.CustomExecutorResources
	rezzes := make(Resources)
	rezzes["cpus"] = fmt.Sprintf("%d", singRez.Cpus)
	rezzes["memory"] = fmt.Sprintf("%d", singRez.MemoryMb)
	rezzes["ports"] = fmt.Sprintf("%d", singRez.NumPorts)

	singReq := req.Request
	numinst := int(singReq.Instances)
	owners := singReq.Owners

	// Singularity's RequestType is an enum of
	//   SERVICE, WORKER, SCHEDULED, ON_DEMAND, RUN_ONCE;
	// Their annotations don't successfully list RequestType into their swagger.
	// Need an approach to handling the type -
	//   either manual Go structs that the resolving context is informed of OR
	//   additional processing of the swagger JSON files before we process it
	kind := ManifestKind("service")

	// We should probably check what kind of Container it is...
	// Which has a similar problem to RequestType - another Java enum that isn't Swaggered
	imageName := singDep.ContainerInfo.Docker.Image

	labels, err := cl.LabelsForImageName(imageName)
	if err != nil {
		return Deployment{}, err
	}

	sv, err := buildSourceVersion(labels)
	if err != nil {
		return Deployment{}, err
	}

	return Deployment{
		Cluster: cluster,
		Owners:  owners,
		Kind:    kind,

		DeployConfig: DeployConfig{
			Resources:    rezzes,
			Env:          env,
			NumInstances: numinst,
		},

		SourceVersion: sv,
	}, nil
}

func buildSourceVersion(labels map[string]string) (SourceVersion, error) {
	missingLabels := make([]string, 0, 3)
	repo, present := labels[DockerRepoLabel]
	if !present {
		missingLabels = append(missingLabels, DockerRepoLabel)
	}

	versionStr, present := labels[DockerVersionLabel]
	if !present {
		missingLabels = append(missingLabels, DockerVersionLabel)
	}

	revision, present := labels[DockerRevisionLabel]
	if !present {
		missingLabels = append(missingLabels, DockerRevisionLabel)
	}

	path, present := labels[DockerPathLabel]
	if !present {
		missingLabels = append(missingLabels, DockerPathLabel)
	}

	if len(missingLabels) > 0 {
		err := fmt.Errorf("Missing labels on manifest for %v", missingLabels)
		return SourceVersion{}, err
	}

	version, err := semv.Parse(versionStr)
	version.Meta = revision

	return SourceVersion{
		RepoURL:    RepoURL(repo),
		Version:    version,
		RepoOffset: RepoOffset(path),
	}, err
}
