package sous

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"

	"github.com/opentable/singularity"
	"github.com/opentable/singularity/dtos"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/samsalisbury/semv"
)

const ReqsPerServer = 10

type (
	sDeploy    *dtos.SingularityDeploy
	sRequest   *dtos.SingularityRequest
	sDepMarker *dtos.SingularityDeployMarker

	underConstruction struct {
		target    Deployment
		depMarker sDepMarker
		deploy    sDeploy
		request   sRequest
	}

	singReq struct {
		sourceUrl string
		sing      *singularity.Client
		reqParent *dtos.SingularityRequestParent
	}
)

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
	sings := make(map[string]*singularity.Client)

	reqCh := make(chan singReq, len(singUrls)*ReqsPerServer)
	depCh := make(chan Deployment, ReqsPerServer)
	defer close(depCh)

	regClient := docker_registry.NewClient()
	regClient.BecomeFoolishlyTrusting()
	defer regClient.Cancel()

	var singWait, depWait sync.WaitGroup

	singWait.Add(len(singUrls))
	for _, url := range singUrls {
		sing := singularity.NewClient(url)
		sings[url] = sing
		go singPipeline(sing, &singWait, reqCh, errCh)
	}

	depWait.Add(1)
	go depPipeline(&depWait, regClient, reqCh, depCh, errCh)

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
		log.Printf("from = %s err = %+v\n", from, err)
		log.Printf("debug.Stack() = %+v\n", string(debug.Stack()))
		switch err := err.(type) {
		default:
			if err != nil {
				errs <- fmt.Errorf("Panicked with not-error: %v", err)
			}
		case error:
			errs <- fmt.Errorf("at %s: %v", from, err)
		}
	}
}

func singPipeline(client *singularity.Client, wg *sync.WaitGroup, reqs chan singReq, errs chan error) {
	defer wg.Done()
	defer catchAndSend(fmt.Sprintf("get requests: %s", client), errs)
	rs, err := getRequestsFromSingularity(client)
	if err != nil {
		errs <- err
		return
	}
	for _, r := range rs {
		reqs <- r
	}
}

func depPipeline(wait *sync.WaitGroup, cl *docker_registry.Client, reqCh chan singReq, depCh chan Deployment, errCh chan error) {
	defer catchAndSend("dependency building", errCh)
	for req := range reqCh {
		wait.Add(1)
		go func(cl *docker_registry.Client, req singReq) {
			defer catchAndSend(fmt.Sprintf("dep from req %s", req.sourceUrl), errCh)

			dep, err := assembleDeployment(cl, req)

			if err != nil {
				errCh <- err
			}

			depCh <- dep
		}(cl, req)
	}
	wait.Done()
}

func assembleDeployment(cl *docker_registry.Client, req singReq) (Deployment, error) {
	rp := req.reqParent
	sing := req.sing

	uc := underConstruction{
		target: Deployment{
			Cluster: req.sourceUrl,
		},
		request: rp.Request,
	}

	rds := rp.RequestDeployState
	if rds == nil {
		return Deployment{}, fmt.Errorf("Singularity response didn't include a deploy state")
	}
	uc.depMarker = rds.PendingDeploy
	if uc.depMarker == nil {
		uc.depMarker = rds.ActiveDeploy
	}
	if uc.depMarker == nil {
		return Deployment{}, fmt.Errorf("Singularity deploy state included no dep markers")
	}

	dh, err := sing.GetDeploy(uc.depMarker.RequestId, uc.depMarker.DeployId) // !!! makes HTTP req
	if err != nil {
		return Deployment{}, err
	}

	uc.deploy = dh.Deploy
	if uc.deploy == nil {
		return Deployment{}, fmt.Errorf("Singularity deploy history included no deploy")
	}

	uc.target.Env = uc.deploy.Env
	if uc.target.Env == nil {
		uc.target.Env = make(map[string]string)
	}

	singRez := uc.deploy.Resources
	uc.target.Resources = make(Resources)
	uc.target.Resources["cpus"] = fmt.Sprintf("%f", singRez.Cpus)
	uc.target.Resources["memory"] = fmt.Sprintf("%f", singRez.MemoryMb)
	uc.target.Resources["ports"] = fmt.Sprintf("%d", singRez.NumPorts)

	uc.target.NumInstances = int(uc.request.Instances)
	uc.target.Owners = uc.request.Owners

	switch uc.request.RequestType {
	default:
		return Deployment{}, fmt.Errorf("Unrecognized response tupe returned by Singularlity: %v", uc.request.RequestType)
	case dtos.SingularityRequestRequestTypeSERVICE:
		uc.target.Kind = ManifestKindService
	case dtos.SingularityRequestRequestTypeWORKER:
		uc.target.Kind = ManifestKindWorker
	case dtos.SingularityRequestRequestTypeON_DEMAND:
		uc.target.Kind = ManifestKindOnDemand
	case dtos.SingularityRequestRequestTypeSCHEDULED:
		uc.target.Kind = ManifestKindScheduled
	case dtos.SingularityRequestRequestTypeRUN_ONCE:
		uc.target.Kind = ManifestKindOnce
	}

	ci := uc.deploy.ContainerInfo
	if ci.Type != dtos.SingularityContainerInfoSingularityContainerTypeDOCKER {
		return Deployment{}, fmt.Errorf("Singularity container isn't a docker container")
	}
	dkr := ci.Docker
	if dkr == nil {
		return Deployment{}, fmt.Errorf("Singularity deploy didn't include a docker info")
	}

	imageName := dkr.Image

	labels, err := cl.LabelsForImageName(imageName) // !!! HTTP request
	if err != nil {
		return Deployment{}, err
	}

	uc.target.SourceVersion, err = buildSourceVersion(labels)
	if err != nil {
		return Deployment{}, err
	}

	return uc.target, nil
}

func getRequestsFromSingularity(client *singularity.Client) ([]singReq, error) {
	singRequests, err := client.GetRequests()
	if err != nil {
		return nil, err
	}

	reqs := make([]singReq, 0, len(singRequests))
	for _, sr := range singRequests {
		reqs = append(reqs, singReq{client.BaseUrl, client, sr})
	}

	return reqs, nil
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
