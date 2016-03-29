package core

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/opentable/sous/tools/cli"
	"github.com/opentable/sous/tools/docker"
	"github.com/opentable/sous/tools/file"
)

// Target describes a buildable Docker file that performs a particular task related to building
// testing and deploying the application. Each pack under packs/ will customise its targets for
// the specific jobs that need to be performed for that pack.
type Target interface {
	String() string
	// Name of the target, as used in command-line operations.
	Name() string
	// GenericDesc is a generic description of the target, applicable to any pack. It is
	// used mostly for help and exposition.
	//GenericDesc() string
	// DependsOn lists the direct dependencies of this target. Dependencies listed as "optional" will
	// always be built when available, but if they are not available will be ignored. It is the job
	// of each package under packs/ to correctly define these relationships.
	DependsOn() []Target
	// Desc is a description of what this target does exactly in the context
	// of the pack that owns it. It should be set by the pack when it is initialised.
	Desc() string
	// Check is a function which tries to detect if this target is possible with the
	// current project. If not, it should return an error.
	Check() error
	// Dockerfile is the shebang method which writes out a functionally complete *docker.Dockerfile
	// This method is only invoked only once the Detect func has successfully detected target availability.
	Dockerfile(*TargetContext) *docker.File
	// Pack is the pack this Target belongs to.
	//Pack() Pack
}

// ContainerTarget is a specialisation of Target that in addition to building a Dockerfile,
// also returns a Docker run command that can be invoked on images built from that Dockerfile, which
// the build process invokes to create a Docker container when needed.
type ContainerTarget interface {
	Target
	// DockerRun returns a Docker run command which the build process can use to
	// create the container.
	DockerRun(*TargetContext) *docker.Run
}

type TargetBase struct {
	name,
	genericDesc string
	pack Pack
}

func (t *TargetBase) Name() string {
	return t.name
}

func (t *TargetBase) GenericDesc() string {
	return t.genericDesc
}

func (t *TargetBase) String() string {
	return t.Name()
}

func (t *TargetBase) Pack() Pack {
	return t.pack
}

type Targets map[string]Target

func (ts Targets) Add(target Target) {
	n := target.Name()
	if _, ok := ts[n]; ok {
		cli.Fatalf("target %s already added", n)
	}
	_, ok := knownTargets[n]
	if !ok {
		cli.Fatalf("target %s is not known", n)
	}
	ts[n] = target
}

// KnownTargets returns a list of all allowed targets along with their generic descriptions.
func KnownTargets() map[string]TargetBase {
	return knownTargets
}

// MustGetTargetBase returns a pointer to a new copy of a known target base,
// or causes the program to fail if the named target does not exist.
func MustGetTargetBase(name string, pack Pack) *TargetBase {
	b, ok := knownTargets[name]
	if !ok {
		cli.Fatalf("target %s not known", name)
	}
	targetCopy := b
	targetCopy.pack = pack
	return &targetCopy
}

type ImageIsStaler interface {
	ImageIsStale(*Context) (bool, string)
}

type PreDockerBuilder interface {
	PreDockerBuild(*Context)
}

type SetStater interface {
	SetState(string, interface{})
}

type Stater interface {
	State() interface{}
}

// RunTarget is used to run the top-level target from build commands, it returns
// a bool: true if the build took place, false if not. The build will not happen
// unless there are relevant changes since the last build. It also returns a state
// object, which can be anything the target decides. This is used in general to allow
// targets in a build chain to communicate with each other, things like the location
// of artifacts built by a target go here, for example.
func (s *Sous) RunTarget(tc *TargetContext) (bool, interface{}) {
	if !tc.ChangesSinceLastBuild().Any() {
		if !s.Flags.ForceRebuild {
			cli.Logf("No changes since last build.")
			cli.Logf("TIP: use -rebuild to rebuild anyway, or -rebuild-all to rebuild all dependencies")
			return false, nil
		}
	}
	cli.Logf(`** ===> Building top-level target "%s"**`, tc.Name())
	return s.runTarget(tc, false)
}

func (s *Sous) runTarget(tc *TargetContext, asDependency bool) (bool, interface{}) {
	depsRebuilt := []string{}
	deps := tc.DependsOn()
	if len(deps) != 0 {
		for _, d := range deps {
			cli.Logf("** ===> Building dependency \"%s\"**", d.Name())
			depTC := s.TargetContext(d.Name())
			depRebuilt, state := s.runTarget(depTC, true)
			if depRebuilt {
				depsRebuilt = append(depsRebuilt, d.Name())
			}
			if ss, ok := tc.Target.(SetStater); ok {
				ss.SetState(depTC.Name(), state)
			}
		}
		cli.Logf("** ===> All dependencies of %s built**", tc.Name())
	}
	// Now we have run all dependencies, run this
	// one if necessary...
	rebuilt := s.buildImageIfNecessary(tc, asDependency, depsRebuilt)
	// If this is a dep and target specifies a docker container, invoke it.
	if asDependency {
		if ct, ok := tc.Target.(ContainerTarget); ok {
			//cli.Logf("** ===> Running target image \"%s\"**", t.Name())
			run, _ := s.RunContainerTarget(ct, tc, rebuilt)
			if run.ExitCode() != 0 {
				cli.Fatalf("** =x=> Docker run failed.**")
			}
		}
	}
	// Get any available state...
	var state interface{}
	if s, ok := tc.Target.(Stater); ok {
		state = s.State()
	}
	return rebuilt, state
}

func (s *Sous) RunContainerTarget(t ContainerTarget, tc *TargetContext, imageRebuilt bool) (*docker.Run, bool) {
	stale, reason, container := s.NewContainerNeeded(tc, imageRebuilt)
	if stale {
		cli.Logf("** ===> Creating new %s container because %s**", t.Name(), reason)
		if container != nil {
			cli.Logf("Force-removing old container %s", container)
			if err := container.ForceRemove(); err != nil {
				cli.Fatalf("Unable to remove outdated container %s; %s", container, err)
			}
		}
		run := t.DockerRun(tc)
		run.Name = containerName(tc)
		return run, true
	}
	cli.Logf("** ===> Re-using build container %s**", container)
	return docker.NewReRun(container), false
}

func containerName(tc *TargetContext) string {
	return fmt.Sprintf("%s-%s", tc.CanonicalPackageName(), tc.Target.Name())
}

func (s *Sous) NewContainerNeeded(tc *TargetContext, imageRebuilt bool) (bool, string, docker.Container) {
	containerName := containerName(tc)
	container := docker.ContainerWithName(containerName)
	if !container.Exists() {
		container = nil
	}

	if container == nil {
		return true, fmt.Sprintf("no container named %s exists", containerName), nil
	}

	if imageRebuilt {
		return true, "its underlying image was rebuilt", container
	}
	// TODO: Check this is comparing the correct images.
	baseImage := tc.Dockerfile().From
	if docker.BaseImageUpdated(baseImage, container.Image()) {
		return true, fmt.Sprintf("base image %s updated", baseImage), container
	}

	return false, "", container
}

// BuildImageIfNecessary usually rebuilds any target if anything of the following
// are true:
//
// - No build is available at all
// - Any files in the working tree have changed
// - Sous has been updated
// - Sous config has changed
//
// However, you may override this behaviour for a specific target by implementing
// the Staler interface: { Stale(*Context) bool }
func (s *Sous) BuildImageIfNecessary(tc *TargetContext) bool {
	return s.buildImageIfNecessary(tc, false, []string{})
}

func (s *Sous) buildImageIfNecessary(tc *TargetContext, asDependency bool, depsRebuilt []string) bool {
	stale, reason := s.needsToBuildNewImage(tc, asDependency, depsRebuilt)
	if !stale {
		return false
	}
	cli.Logf("** ===> Rebuilding image for %s because %s**", tc.Name(), reason)
	s.BuildImage(tc)
	return true
}

func (s *Sous) NeedsToBuildNewImage(tc *TargetContext, asDependency bool) (bool, string) {
	return s.needsToBuildNewImage(tc, asDependency, []string{})
}

// NeedsBuild detects if the project's last
// build is stale, and if it therefore needs to be rebuilt. This can be overidden
// by implementing the Staler interfact on individual build targets. This default
// implementation rebuilds on absolutely any change in sous (i.e. new version/new
// config) or in the working tree (new or modified files).
func (s *Sous) needsToBuildNewImage(tc *TargetContext, asDependency bool, depsRebuilt []string) (bool, string) {
	t := tc.Target
	c := tc.Context
	if len(depsRebuilt) == 1 {
		return true, fmt.Sprintf("its %s dependency was rebuilt", depsRebuilt[0])
	}
	if len(depsRebuilt) > 1 {
		return true, fmt.Sprintf("its dependencies [%s] were reubilt", strings.Join(depsRebuilt, ", "))
	}
	if s.Flags.ForceRebuildAll {
		return true, "-rebuild-all flag was used"
	}
	if s.Flags.ForceRebuild && !asDependency {
		return true, "-rebuild flag was used"
	}
	changes := tc.ChangesSinceLastBuild()
	if staler, ok := t.(ImageIsStaler); ok {
		if stale, reason := staler.ImageIsStale(c); stale {
			return true, reason
		}
	} else if changes.Any() {
		reason := "changes were detected"
		if changes.WorkingTreeChanged {
			reason = "your working tree has changed"
		} else if changes.NewCommit {
			reason = "you have a new commit"
		} else if changes.NoBuiltImage {
			reason = "no corresponding image exists yet"
		} else if changes.SousUpdated {
			reason = "sous itself was updated"
		}
		return true, reason
	}
	// Always force a rebuild if is base image has been updated.
	baseImage := tc.Dockerfile().From
	// TODO: This is probably a bit too aggressive, consider only asking the user to
	// update base images every 24 hours, if they have actually been updated.
	s.UpdateBaseImage(baseImage)
	if tc.BuildNumber() == 1 {
		return true, fmt.Sprintf("there are no successful builds yet for the current revision (%s)", c.Git.CommitSHA)
	}
	if !tc.LastBuildImageExists() {
		return true, fmt.Sprintf("the last successful build image no longer exists (%s)", tc.PrevDockerTag())
	}
	if docker.BaseImageUpdated(baseImage, tc.PrevDockerTag()) {
		return true, fmt.Sprintf("the base image %s was updated", baseImage)
	}
	// Always force a build if Sous itself has been updated
	// NB: Always keep this check until last, since it's annoying, so only report this as the reason to rebuild
	// if none of the reasons above hold true. Sous does its own PR innit.
	if changes.SousUpdated {
		return true, fmt.Sprintf("sous itself or its config was updated")
	}
	return false, ""
}

func (s *Sous) BuildImage(tc *TargetContext) {
	tc.IncrementBuildNumber()
	if file.Exists("Dockerfile") {
		cli.Warn("./Dockerfile ignored by sous; use `sous dockerfile %s` to see the Dockerfile in effect", tc.Name())
	}
	if prebuilder, ok := tc.Target.(PreDockerBuilder); ok {
		prebuilder.PreDockerBuild(tc.Context)
	}
	// NB: Always rebuild the Dockerfile after running pre-build, since pre-build
	// may update target state to reflect things like copied file locations etc.
	tc.SaveFile(s.Dockerfile(tc).String(), "Dockerfile")
	docker.BuildFile(tc.FilePath("Dockerfile"), ".", tc.DockerTag())
	tc.Commit()
}

// Sous.Dockerfile is the canonical source for all Dockerfiles. It takes
// the Dockerfile defined by the pack target, and decorates it with additional
// metadata.
func (s *Sous) Dockerfile(tc *TargetContext) *docker.File {
	df := tc.Target.Dockerfile(tc)
	df.Maintainer = tc.User
	p := s.Config.DockerLabelPrefix + "."
	df.LABEL(map[string]string{
		p + "build.number":            strconv.Itoa(tc.BuildNumber()),
		p + "build.pack.name":         tc.Buildpack.Name,
		p + "build.pack.id":           strings.ToLower(tc.Buildpack.Name),
		p + "build.target":            tc.Target.Name(),
		p + "build.tool.name":         "sous",
		p + "build.tool.version":      sous.Version,
		p + "build.tool.revision":     sous.Revision,
		p + "build.tool.os":           sous.OS,
		p + "build.tool.arch":         sous.Arch,
		p + "build.machine.host":      tc.Host,
		p + "build.machine.fullhost":  tc.FullHost,
		p + "build.user":              tc.User,
		p + "build.source.repository": tc.Git.CanonicalRepoName(),
		p + "build.source.revision":   tc.Git.CommitSHA,
		p + "build.package.name":      tc.CanonicalPackageName(),
		p + "build.package.version":   tc.BuildVersion.String(),
	})
	return df
}

var knownTargets = map[string]TargetBase{
	"compile": TargetBase{
		name:        "compile",
		genericDesc: "a container that performs any pre-compile steps that should happen before building the app for deployment",
	},
	"app": TargetBase{
		name:        "app",
		genericDesc: "a container containing the application itself, as it would be deployed",
	},
	"test": TargetBase{
		name:        "test",
		genericDesc: "a container whose only job is to run unit tests and then exit",
	},
	"smoke": TargetBase{
		name:        "smoke",
		genericDesc: "a container whose only job is to run (smoke) tests against a remote instance of this app",
	},
}
