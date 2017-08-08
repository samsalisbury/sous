package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/davecgh/go-spew/spew"
	sous "github.com/opentable/sous/lib"
)

type splitBuilder struct {
	context          *sous.BuildContext
	detected         *sous.DetectResult
	VersionConfig    string
	RevisionConfig   string
	buildImageID     string
	buildContainerID string
	tempDir          string
	buildDir         string
	RunSpec          *MultiImageRunSpec
	subBuilders      []*runnableBuilder
}

func (sb *splitBuilder) buildBuild() error {
	offset := sb.context.Source.OffsetDir
	if offset == "" {
		offset = "."
	}

	v := sb.context.Version().Version
	v.Meta = ""
	sb.VersionConfig = fmt.Sprintf("%s=%s", AppVersionBuildArg, v)
	sb.RevisionConfig = fmt.Sprintf("%s=%s", AppRevisionBuildArg, sb.context.Version().RevID())

	cmd := []interface{}{"build", "--pull"}
	r := sb.detected.Data.(detectData)
	if r.HasAppVersionArg {
		cmd = append(cmd, "--build-arg", sb.VersionConfig)
	}
	if r.HasAppRevisionArg {
		cmd = append(cmd, "--build-arg", sb.RevisionConfig)
	}

	cmd = append(cmd, offset)

	output, err := sb.context.Sh.Stdout("docker", cmd...)
	if err != nil {
		return err
	}

	spew.Dump(output)
	match := successfulBuildRE.FindStringSubmatch(string(output))
	if match == nil {
		return fmt.Errorf("Couldn't find container id in:\n%s", output)
	}
	sb.buildImageID = match[1]

	return nil
}

func (sb *splitBuilder) setupTempdir() error {
	dir, err := ioutil.TempDir("", "sous-split-build")
	if err != nil {
		return err
	}
	sb.tempDir = dir
	sb.buildDir = filepath.Join(sb.tempDir, "build")
	return os.MkdirAll(sb.buildDir, os.ModePerm)
}

func (sb *splitBuilder) createBuildContainer() error {
	output, err := sb.context.Sh.Stdout("docker", "create", sb.buildImageID)
	if err != nil {
		return err
	}
	sb.buildContainerID = strings.TrimSpace(output)

	return nil
}

func (sb *splitBuilder) extractRunSpec() error {
	runspecPath := sb.detected.Data.(detectData).RunImageSpecPath
	destPath := filepath.Join(sb.tempDir, "runspec.json")
	_, err := sb.context.Sh.Stdout("docker", "cp", fmt.Sprintf("%s:%s", sb.buildContainerID, runspecPath), destPath)
	if err != nil {
		return err
	}

	specF, err := os.Open(destPath)
	if err != nil {
		return err
	}

	sb.RunSpec = &MultiImageRunSpec{}
	dec := json.NewDecoder(specF)
	return dec.Decode(sb.RunSpec)
}

func (sb *splitBuilder) validateRunSpec() error {
	flaws := sb.RunSpec.Validate()
	if len(flaws) > 0 {
		msg := "Deploy spec invalid:"
		for _, f := range flaws {
			msg += "\n\t" + f.Repair().Error()
		}
		return errors.New(msg)
	}
	return nil
}

func (sb *splitBuilder) constructImageBuilders() error {
	rs := sb.RunSpec.Normalized()
	sb.subBuilders = []*runnableBuilder{}

	for _, spec := range rs.Images {
		sb.subBuilders = append(sb.subBuilders, &runnableBuilder{
			RunSpec:      spec,
			splitBuilder: sb,
		})
	}

	return nil
}

func (sb *splitBuilder) eachBuilder(f func(*runnableBuilder) error) error {
	for _, rb := range sb.subBuilders {
		err := f(rb)
		if err != nil {
			return err
		}
	}

	return nil
}

func (sb *splitBuilder) extractFiles() error {
	return sb.eachBuilder((*runnableBuilder).extractFiles)
}

// xxx consider simply *not* tearing down, and reusing if existant, possibly with an option.
func (sb *splitBuilder) teardownBuildContainer() error {
	_, err := sb.context.Sh.Stdout("docker", "rm", sb.buildContainerID)
	return err
}

func (sb *splitBuilder) templateDockerfiles() error {
	return sb.eachBuilder((*runnableBuilder).templateDockerfile)
}

func (sb *splitBuilder) buildRunnables() error {
	return sb.eachBuilder((*runnableBuilder).build)
}
