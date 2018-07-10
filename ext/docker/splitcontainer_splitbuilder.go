package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	sous "github.com/opentable/sous/lib"
)

type splitBuilder struct {
	context          *sous.BuildContext
	detected         *sous.DetectResult
	start            time.Time
	VersionConfig    string
	RevisionConfig   string
	buildImageID     string
	buildContainerID string
	tempDir          string
	buildDir         string
	RunSpec          *MultiImageRunSpec
	subBuilders      []*runnableBuilder
}

func (sb *splitBuilder) versionName() string {
	v := sb.context.Version().Version
	v.Meta = ""
	return v.String()
}

func (sb *splitBuilder) revisionName() string {
	return sb.context.RevID()
}

func (sb *splitBuilder) versionConfig() string {
	return fmt.Sprintf("%s=%s", AppVersionBuildArg, sb.versionName())
}

func (sb *splitBuilder) revisionConfig() string {
	return fmt.Sprintf("%s=%s", AppRevisionBuildArg, sb.revisionName())
}

func (sb *splitBuilder) begin() error {
	sb.start = time.Now()
	return nil
}

func (sb *splitBuilder) buildBuild() error {
	offset := sb.context.Source.OffsetDir
	if offset == "" {
		offset = "."
	}

	cmd := []interface{}{"build"}
	if !sb.context.Source.DevBuild {
		//pull the image if you aren't doing a dev build.
		cmd = append(cmd, "--pull")
	}

	r := sb.detected.Data.(detectData)
	if r.HasAppVersionArg {
		cmd = append(cmd, "--build-arg", sb.versionConfig())
	}
	if r.HasAppRevisionArg {
		cmd = append(cmd, "--build-arg", sb.revisionConfig())
	}

	itag := intermediateTag()
	cmd = append(cmd, "-t", itag)

	// XXX I really think this should be "-f", path.Join(offset, "Dockerfile") -jdl
	cmd = append(cmd, offset)

	_, err := sb.context.Sh.Stdout("docker", cmd...)
	if err != nil {
		return err
	}

	sb.buildImageID = itag

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

func (sb *splitBuilder) result() *sous.BuildResult {
	return &sous.BuildResult{
		Elapsed: time.Since(sb.start),
		Products: append(
			sb.products(),
			&sous.BuildProduct{ID: sb.buildImageID, Kind: "builder",
				Advisories: append(sb.context.Advisories, sous.IsBuilder, sous.NotService)}),
	}
}

func (sb *splitBuilder) products() (ps []*sous.BuildProduct) {
	sb.eachBuilder(func(b *runnableBuilder) error {
		ps = append(ps, b.product())
		return nil
	})
	return
}
