//+build smoke

package smoke

import (
	"fmt"
	"strings"
	"testing"

	"github.com/opentable/sous/util/filemap"
)

func dockerBuildAddArtifactInit(t *testing.T, f *fixture, client *sousProject, flags *sousFlags, transforms ...ManifestTransform) (dockerRef string) {
	t.Helper()

	dockerRef = dockerBuildAddArtifact(t, f, client, flags)

	client.MustRun(t, "init", flags.SousInitFlags(), "-use-otpl-deploy")
	client.TransformManifest(t, flags, transforms...)

	return dockerRef
}

func dockerBuildAddArtifact(t *testing.T, f *fixture, client *sousProject, flags *sousFlags) (dockerRef string) {
	t.Helper()
	tag := flags.tag
	if tag == "" {
		t.Fatalf("you must add a non-empty tag flag")
	}
	reg := client.Config.Docker.RegistryHost
	repo := flags.repo
	dockerTag := f.IsolatedVersionTag(tag)
	dockerRepo := fmt.Sprintf("%s/%s", reg, repo)
	dockerRef = strings.ToLower(fmt.Sprintf("%s:%s", dockerRepo, dockerTag))

	mustDoCMD(t, client.Dir, "docker", "build", "-t", dockerRef, ".")
	mustDoCMD(t, client.Dir, "docker", "push", dockerRef)

	client.MustRun(t, "artifact add", flags.SourceIDFlags(), "-docker-image", dockerRepo)

	return dockerRef
}

func TestOTPL(t *testing.T) {

	// FixedDimension is because otpl deploy can only work with simple dockerfile
	// projects, not split build projects.
	pf := newRunner(t, matrix().FixedDimension("project", "simple"))

	pf.Run("artifact-add", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{tag: "1.2.3", repo: p.repo}

		dockerRef := dockerBuildAddArtifact(t, f, p, flags)

		output := p.MustRun(t, "artifact get", flags)

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
		p := setupProject(t, f, f.Projects.HTTPServer().Merge(
			makeOTPLConfig(reqID, cluster)))

		flags := &sousFlags{
			kind:    "http-service",
			repo:    p.repo,
			tag:     "1.2.3",
			cluster: "cluster1",
		}

		dockerBuildAddArtifactInit(t, f, p, flags, setMinimalMemAndCPUNumInst1)

		m := p.getManifest(t, flags)
		d := assertManifestExactlyOneDeployment(t, m, cluster)

		if got := d.SingularityRequestID; got != reqID {
			t.Fatalf("got sing req id %q; want %q", got, reqID)
		}

		p.MustRun(t, "deploy", flags.SousDeployFlags())

		assertActiveStatus(t, f, reqID)
		assertSingularityRequestTypeService(t, f, reqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
	})

	pf.Run("root-withflavor", func(t *testing.T, f *fixture) {
		reqID := f.IsolatedRequestID("request1")
		cluster := f.IsolatedClusterName("cluster1")
		p := setupProject(t, f, f.Projects.HTTPServer().Merge(
			makeOTPLConfig(reqID, cluster+".flavor1")))

		flags := &sousFlags{
			flavor:  "flavor1",
			kind:    "http-service",
			repo:    p.repo,
			tag:     "1.2.3",
			cluster: "cluster1",
		}

		dockerBuildAddArtifactInit(t, f, p, flags, setMinimalMemAndCPUNumInst1)

		m := p.getManifest(t, flags)
		d := assertManifestExactlyOneDeployment(t, m, cluster)

		if got := d.SingularityRequestID; got != reqID {
			t.Fatalf("got sing req id %q; want %q", got, reqID)
		}

		p.MustRun(t, "deploy", flags.SousDeployFlags())

		assertActiveStatus(t, f, reqID)
		assertSingularityRequestTypeService(t, f, reqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
	})

	pf.Run("root-withoffset-flag", func(t *testing.T, f *fixture) {
		reqID := f.IsolatedRequestID("request1")
		cluster := f.IsolatedClusterName("cluster1")
		p := setupProject(t, f,
			filemap.Merge(
				f.Projects.HTTPServer(),
				makeOTPLConfig(reqID, cluster),
			).PrefixAll("offset1"),
		)

		p.Dir += "/offset1"

		flags := &sousFlags{
			kind:    "http-service",
			repo:    p.repo,
			tag:     "1.2.3",
			cluster: "cluster1",
			offset:  "offset1",
		}

		dockerBuildAddArtifactInit(t, f, p, flags, setMinimalMemAndCPUNumInst1)

		m := p.getManifest(t, flags)
		if got := m.Source.Dir; got != "offset1" {
			t.Fatalf("got offset %q; want %q", got, "offset1")
		}

		d := assertManifestExactlyOneDeployment(t, m, cluster)

		if got := d.SingularityRequestID; got != reqID {
			t.Fatalf("got sing req id %q; want %q", got, reqID)
		}

		p.MustRun(t, "deploy", flags.SousDeployFlags())

		assertActiveStatus(t, f, reqID)
		assertSingularityRequestTypeService(t, f, reqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
	})

	pf.Run("root-withoffset-noflag", func(t *testing.T, f *fixture) {

		f.KnownToFailHere(t)

		reqID := f.IsolatedRequestID("request1")
		cluster := f.IsolatedClusterName("cluster1")
		p := setupProject(t, f,
			filemap.Merge(
				f.Projects.HTTPServer(),
				makeOTPLConfig(reqID, cluster),
			).PrefixAll("offset1"),
		)

		p.Dir += "/offset1"

		flags := &sousFlags{
			kind:    "http-service",
			repo:    p.repo,
			tag:     "1.2.3",
			cluster: "cluster1",
		}

		dockerBuildAddArtifactInit(t, f, p, flags, setMinimalMemAndCPUNumInst1)

		m := p.getManifest(t, flags)
		if got := m.Source.Dir; got != "offset1" {
			t.Fatalf("got offset %q; want %q", got, "offset1")
		}

		d := assertManifestExactlyOneDeployment(t, m, cluster)

		if got := d.SingularityRequestID; got != reqID {
			t.Fatalf("got sing req id %q; want %q", got, reqID)
		}

		p.MustRun(t, "deploy", flags.SousDeployFlags())

		assertActiveStatus(t, f, reqID)
		assertSingularityRequestTypeService(t, f, reqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
	})

	pf.Run("fail-init-unknown-fields-req", func(t *testing.T, f *fixture) {
		reqID := f.IsolatedRequestID("request1")
		cluster := f.IsolatedClusterName("cluster1")
		otplConfig := makeOTPLConfig(reqID, cluster, func(req, dep *interface{}) {
			reqMap := (*req).(map[string]interface{})
			reqMap["invalidfield1"] = 0
			*req = reqMap
		})
		p := setupProject(t, f, filemap.Merge(
			f.Projects.HTTPServer(),
			otplConfig,
		))

		flags := &sousFlags{
			kind:    "http-service",
			repo:    p.repo,
			tag:     "1.2.3",
			cluster: "cluster1",
		}

		p.MustFail(t, "init", flags.SousInitFlags(), "-use-otpl-deploy")
	})

	pf.Run("fail-init-unknown-fields-dep", func(t *testing.T, f *fixture) {
		reqID := f.IsolatedRequestID("request1")
		cluster := f.IsolatedClusterName("cluster1")
		otplConfig := makeOTPLConfig(reqID, cluster, func(req, dep *interface{}) {
			depMap := (*dep).(map[string]interface{})
			depMap["invalidfield1"] = 0
			*dep = depMap
		})
		p := setupProject(t, f, filemap.Merge(
			f.Projects.HTTPServer(),
			otplConfig,
		))

		flags := &sousFlags{
			kind:    "http-service",
			repo:    p.repo,
			tag:     "1.2.3",
			cluster: "cluster1",
		}

		p.MustFail(t, "init", flags.SousInitFlags(), "-use-otpl-deploy")
	})
}
