//+build smoke

package smoke

import (
	"strings"
	"testing"

	sous "github.com/opentable/sous/lib"
)

func initProject(t *testing.T, client *sousProject, flags *sousFlags, transforms ...ManifestTransform) {
	client.MustRun(t, "init", flags.SousInitFlags())
	client.TransformManifest(t, flags.ManifestIDFlags(), transforms...)
}

// initBuild is a macro to fast-forward to tests to a point where we have
// initialised the project, transformed the manifest, and performed a build.
func initBuild(t *testing.T, client *sousProject, flags *sousFlags, transforms ...ManifestTransform) {
	initProject(t, client, flags, transforms...)
	client.MustRun(t, "build", flags.SourceIDFlags())
}

// initBuildDeploy is a macro for fast-forwarding tests to a point where we have
// initialised the project as a kind project, built it with -tag tag and
// deployed that tag successfully to cluster.
func initBuildDeploy(t *testing.T, client *sousProject, flags *sousFlags, transforms ...ManifestTransform) {
	initBuild(t, client, flags, transforms...)
	client.MustRun(t, "deploy", flags.SousDeployFlags())
}

func TestSmoke(t *testing.T) {

	m := newRunner(t, matrix())

	m.Run("simple", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{kind: "http-service", tag: "1.2.3", cluster: "cluster1", repo: p.repo}
		reqID := f.DefaultSingReqID(t, flags)

		initBuildDeploy(t, p, flags, setMinimalMemAndCPUNumInst1)

		assertActiveStatus(t, f, reqID)
		assertSingularityRequestTypeService(t, f, reqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
	})

	m.Run("fail-zero-instances", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{kind: "http-service", tag: "1.2.3", repo: p.repo}

		initBuild(t, p, flags, setMinimalMemAndCPUNumInst0)

		p.MustFail(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1.2.3")
	})

	m.Run("fail-container-crash", func(t *testing.T, f *fixture) {

		p := setupProject(t, f, f.Projects.Failer())

		flags := &sousFlags{
			kind:    "http-service",
			tag:     "1.2.3",
			cluster: "cluster1",
			repo:    p.repo,
		}

		initBuild(t, p, flags, setMinimalMemAndCPUNumInst1)

		got := p.MustFail(t, "deploy", flags.SousDeployFlags())
		want := `Deploy failure:`
		if !strings.Contains(got, want) {
			t.Fatalf("want stderr to contain %q; got %q", want, got)
		}

		reqID := f.DefaultSingReqID(t, flags)
		assertActiveStatus(t, f, reqID)
		assertSingularityRequestTypeService(t, f, reqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
	})

	m.Run("flavors", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{
			kind: "http-service", tag: "1.2.3", cluster: "cluster1",
			flavor: "flavor1", repo: p.repo,
		}

		initBuildDeploy(t, p, flags, setMinimalMemAndCPUNumInst1)

		reqID := f.DefaultSingReqID(t, flags)
		assertActiveStatus(t, f, reqID)
		assertSingularityRequestTypeService(t, f, reqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
	})

	m.Run("pause-unpause", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{kind: "http-service", tag: "1", cluster: "cluster1", repo: p.repo}

		initBuildDeploy(t, p, flags, setMinimalMemAndCPUNumInst1)

		// Prepare a couple more builds...
		p.MustRun(t, "build", nil, "-tag", "2")
		p.MustRun(t, "build", nil, "-tag", "3")

		reqID := f.DefaultSingReqID(t, flags)
		assertActiveStatus(t, f, reqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, reqID)
		assertSingularityRequestTypeService(t, f, reqID)

		f.Singularity.PauseRequestForDeployment(t, reqID)
		p.MustFail(t, "deploy", nil, "-cluster", "cluster1", "-tag", "2")
		f.Singularity.UnpauseRequestForDeployment(t, reqID)

		p.MustRun(t, "deploy", nil, "-cluster", "cluster1", "-tag", "3")
		assertActiveStatus(t, f, reqID)
	})

	m.Run("scheduled", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.Sleeper())

		flags := &sousFlags{kind: "scheduled", tag: "1.2.3", cluster: "cluster1", repo: p.repo}

		initBuildDeploy(t, p, flags, func(m sous.Manifest) sous.Manifest {
			clusterName := "cluster1" + f.ClusterSuffix
			d := m.Deployments[clusterName]
			d.NumInstances = 1
			d.Schedule = "*/5 * * * *"
			m.Deployments[clusterName] = d

			m.Deployments = setMemAndCPUForAll(m.Deployments)

			return m
		})

		reqID := f.DefaultSingReqID(t, flags)
		assertSingularityRequestTypeScheduled(t, f, reqID)
		assertActiveStatus(t, f, reqID)
		assertNilHealthCheckOnLatestDeploy(t, f, reqID)
	})

	m.Run("custom-reqid-first-deploy", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{kind: "http-service", tag: "1.2.3", cluster: "cluster1", repo: p.repo}

		customID := "some-custom-req-id" + f.ClusterSuffix

		initBuildDeploy(t, p, flags,
			setMinimalMemAndCPUNumInst1,
			p.setSingularityRequestID(t, "cluster1", customID),
		)

		assertSingularityRequestTypeService(t, f, customID)
		assertActiveStatus(t, f, customID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, customID)
	})

	m.Run("custom-reqid-second-deploy", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{kind: "http-service", tag: "1.2.3", cluster: "cluster1", repo: p.repo}

		initBuildDeploy(t, p, flags, setMinimalMemAndCPUNumInst1)

		originalReqID := f.DefaultSingReqID(t, flags)

		assertSingularityRequestTypeService(t, f, originalReqID)
		assertActiveStatus(t, f, originalReqID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, originalReqID)

		customID := "some-custom-req-id" + f.ClusterSuffix
		p.TransformManifest(t, nil, p.setSingularityRequestID(t, "cluster1", customID))

		// Force to avoid having to make another build.
		p.MustRun(t, "deploy", nil, "-force", "-cluster", "cluster1", "-tag", "1.2.3")

		assertSingularityRequestTypeService(t, f, customID)
		assertActiveStatus(t, f, customID)
		assertNonNilHealthCheckOnLatestDeploy(t, f, customID)

		// TODO: Implement cleanup of old request.
		//assertRequestDoesNotExist(t, f, originalReqID)

		assertActiveStatus(t, f, originalReqID)
	})

	m.Run("change-reqid", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{kind: "http-service", tag: "1.2.3", cluster: "cluster1", repo: p.repo}

		customID1 := "some-custom-req-id1" + f.ClusterSuffix

		initBuildDeploy(t, p, flags,
			setMinimalMemAndCPUNumInst1,
			p.setSingularityRequestID(t, "cluster1", customID1),
		)

		assertSingularityRequestTypeService(t, f, customID1)
		assertActiveStatus(t, f, customID1)
		assertNonNilHealthCheckOnLatestDeploy(t, f, customID1)

		customID2 := "some-custom-req-id2" + f.ClusterSuffix

		p.TransformManifest(t, nil, p.setSingularityRequestID(t, "cluster1", customID2))

		p.MustRun(t, "deploy", nil, "-force", "-cluster", "cluster1", "-tag", "1.2.3")

		assertSingularityRequestTypeService(t, f, customID2)
		assertActiveStatus(t, f, customID2)
		assertNonNilHealthCheckOnLatestDeploy(t, f, customID2)

		// TODO: Implement cleanup of old request.
		//assertRequestDoesNotExist(t, f, customID1)

		assertActiveStatus(t, f, customID1) // This works because we do not yet do cleanup.
	})

	m.Run("getartifact-success", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())
		flags := &sousFlags{
			repo: p.repo,
			tag:  "1.2.3",
		}
		p.MustRun(t, "build", flags)
		p.MustRun(t, "artifact get", flags)
	})

	m.Run("repeated-tag-fails-build", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{
			repo: p.repo,
			tag:  "1.2.3",
		}

		p.MustRun(t, "build", flags)

		// This line means we end up with a different Dockerfile, thus a
		// different image digest once it's built.
		// TODO SS: Provide a more explicit mechanism for changing
		// files in a project after initial setup.
		f.Projects.Sleeper().Overwrite(p.Dir)

		gotStderr := p.MustFail(t, "build", flags)
		want := "artifact already registered"
		if !strings.Contains(gotStderr, want) {
			t.Errorf("got stderr %q, want it to contain %q", gotStderr, want)
		}
	})

	m.Run("repeated-tag-fails-build-norepoflag", func(t *testing.T, f *fixture) {
		p := setupProject(t, f, f.Projects.HTTPServer())

		flags := &sousFlags{
			tag: "1.2.3",
		}

		p.MustRun(t, "build", flags)

		// This line means we end up with a different Dockerfile, thus a
		// different image digest once it's built.
		// TODO SS: Provide a more explicit mechanism for changing
		// files in a project after initial setup.
		f.Projects.Sleeper().Overwrite(p.Dir)

		p.MustRun(t, "context", flags)

		gotStderr := p.MustFail(t, "build", flags)
		want := "artifact already registered"
		if !strings.Contains(gotStderr, want) {
			t.Errorf("got stderr %q, want it to contain %q", gotStderr, want)
		}
	})
}
