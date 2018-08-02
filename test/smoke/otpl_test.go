//+build smoke

package smoke

import (
	"fmt"
	"strings"
	"testing"

	"github.com/opentable/sous/util/filemap"
)

func dockerBuildAddArtifactInit(t *testing.T, f *fixture, client *sousClient, flags *sousFlags, transforms ...ManifestTransform) (dockerRef string) {
	t.Helper()

	dockerRef = dockerBuildAddArtifact(t, f, client, flags)

	client.MustRun(t, "init", flags.SousInitFlags(), "-use-otpl-deploy")
	client.TransformManifest(t, flags, transforms...)

	return dockerRef
}

func dockerBuildAddArtifact(t *testing.T, f *fixture, client *sousClient, flags *sousFlags) (dockerRef string) {
	t.Helper()
	tag := flags.tag
	if tag == "" {
		t.Fatalf("you must add a non-empty tag flag")
	}
	reg := client.Config.Docker.RegistryHost
	repo := flags.repo
	dockerTag := f.IsolatedVersionTag(tag)
	dockerRepo := fmt.Sprintf("%s/%s", reg, repo)
	dockerRef = fmt.Sprintf("%s:%s", dockerRepo, dockerTag)

	mustDoCMD(t, client.Dir, "docker", "build", "-t", dockerRef, ".")
	mustDoCMD(t, client.Dir, "docker", "push", dockerRef)

	client.MustRun(t, "artifact add", nil, "-docker-image", dockerRepo, "-repo", repo, "-tag", tag)

	return dockerRef
}

// makeOTPLConfig creates valid otpl-deploy config files using reqID as the
// request ID and envFull as the <env>[.<flavor>] string used by otpl-deploy.
func makeOTPLConfig(reqID, envFull string) filemap.FileMap {
	return filemap.FileMap{
		"config/" + envFull + "/singularity.json": `
				{
					"requestId": "` + reqID + `",
					"resources": {
						"cpus": 0.01,
						"memoryMb": 1,
						"numPorts": 3
					}
				}`,
		"config/" + envFull + "/singularity-request.json": `
				{
					"id": "` + reqID + `",
					"requestType": "SERVICE",
					"owners": [
					    "test-user1@example.com"
					],
					"instances": 3
				}`,
	}
}

func TestOTPL(t *testing.T) {

	// FixedDimension is because otpl deploy can only work with simple dockerfile
	// projects, not split build projects.
	pf := newRunner(t, matrix().FixedDimension("project", "simple"))

	pf.Run("artifact-add", func(t *testing.T, f *fixture) {
		client := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{tag: "1.2.3", repo: "github.com/some-user/project1"}

		dockerRef := dockerBuildAddArtifact(t, f, client, flags)

		output := client.MustRun(t, "artifact get", flags)

		if !strings.Contains(output, dockerRef) {
			// TODO SS: Figure out how to do this assertion given that we do
			// not store the Docker tag sent, only the  digest.
			//t.Errorf("output did not contain %q; was:\n%s", dockerRef, output)
		} else {
			// TODO SS: Remove next line once we have the assertion above.
			t.Logf(output)
		}
	})

	pf.Run("root-noflavor", func(t *testing.T, f *fixture) {
		reqID := f.IsolatedRequestID("request1")
		cluster := f.IsolatedClusterName("cluster1")
		client := setupProject(t, f, f.Projects.HTTPServer().Merge(
			makeOTPLConfig(reqID, cluster)))

		flags := &sousFlags{
			kind:    "http-service",
			repo:    "github.com/user1/project1",
			tag:     "1.2.3",
			cluster: "cluster1",
		}

		dockerBuildAddArtifactInit(t, f, client, flags, setMinimalMemAndCPUNumInst1)

		m := client.getManifest(t, flags)
		d := assertManifestExactlyOneDeployment(t, m, cluster)

		if got := d.SingularityRequestID; got != reqID {
			t.Fatalf("got sing req id %q; want %q", got, reqID)
		}

		client.MustRun(t, "deploy", flags.SousDeployFlags())

		assertActiveStatus(t, f, reqID)
		assertSingularityRequestTypeService(t, f, reqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
	})

	pf.Run("root-withflavor", func(t *testing.T, f *fixture) {
		reqID := f.IsolatedRequestID("request1")
		cluster := f.IsolatedClusterName("cluster1")
		client := setupProject(t, f, f.Projects.HTTPServer().Merge(
			makeOTPLConfig(reqID, cluster+".flavor1")))

		flags := &sousFlags{
			flavor:  "flavor1",
			kind:    "http-service",
			repo:    "github.com/user1/project1",
			tag:     "1.2.3",
			cluster: "cluster1",
		}

		dockerBuildAddArtifactInit(t, f, client, flags, setMinimalMemAndCPUNumInst1)

		m := client.getManifest(t, flags)
		d := assertManifestExactlyOneDeployment(t, m, cluster)

		if got := d.SingularityRequestID; got != reqID {
			t.Fatalf("got sing req id %q; want %q", got, reqID)
		}

		client.MustRun(t, "deploy", flags.SousDeployFlags())

		assertActiveStatus(t, f, reqID)
		assertSingularityRequestTypeService(t, f, reqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
	})

	pf.Run("root-withoffset", func(t *testing.T, f *fixture) {
		reqID := f.IsolatedRequestID("request1")
		cluster := f.IsolatedClusterName("cluster1")
		client := setupProject(t, f,
			filemap.Merge(
				f.Projects.HTTPServer(),
				makeOTPLConfig(reqID, cluster),
			).PrefixAll("offset1"),
		)

		client.Dir = client.Dir + "/offset1"

		flags := &sousFlags{
			kind:    "http-service",
			repo:    "github.com/user1/project1",
			tag:     "1.2.3",
			cluster: "cluster1",
		}

		dockerBuildAddArtifactInit(t, f, client, flags, setMinimalMemAndCPUNumInst1)

		m := client.getManifest(t, flags)
		if got := m.ID().Source.Dir; got != "offset1" {
			t.Fatalf("got offset %q; want %q", got, "offset1")
		}

		d := assertManifestExactlyOneDeployment(t, m, cluster)

		if got := d.SingularityRequestID; got != reqID {
			t.Fatalf("got sing req id %q; want %q", got, reqID)
		}

		client.MustRun(t, "deploy", flags.SousDeployFlags())

		assertActiveStatus(t, f, reqID)
		assertSingularityRequestTypeService(t, f, reqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
	})

	pf.Run("fail-init-unknown-fields", func(t *testing.T, f *fixture) {
		reqID := f.IsolatedRequestID("request1")
		cluster := f.IsolatedClusterName("cluster1")
		client := setupProject(t, f, f.Projects.HTTPServer().Merge(
			makeOTPLConfig(reqID, cluster)))

		flags := &sousFlags{
			kind:    "http-service",
			repo:    "github.com/build-init-deploy-user/project1",
			tag:     "1.2.3",
			cluster: "cluster1",
		}

		client.MustFail(t, "init", flags.SousInitFlags(), "-use-otpl-deploy")
	})
}
