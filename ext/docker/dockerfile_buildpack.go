package docker

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/opentable/sous/lib"
)

// DockerfileBuildpack is a simple buildpack for building projects using
// their own Dockerfile.
type DockerfileBuildpack struct{}

const (
	// AppVersionBuildArg is the name of a docker build argument used to inject
	// the version of the app being built.
	AppVersionBuildArg = "APP_VERSION"
	// AppRevisionBuildArg is the name of a docker build argument used to inject
	// the revision of the app being built.
	AppRevisionBuildArg = "APP_REVISION"
)

var (
	appVersionPattern  = regexp.MustCompile(`(?m)^ARG ` + AppVersionBuildArg + `\b`)
	appRevisionPattern = regexp.MustCompile(`(?m)^ARG ` + AppRevisionBuildArg + `\b`)
)

// datectData is data passed from the detect step to the build step as the
// Data field in the DetectResult.
type detectData struct {
	// HasAppVersionArg is true if the Dockerfile contains a line matching
	// appVersionPattern.
	HasAppVersionArg,
	// HasAppRevisionArg is true if the Dockerfile contains a line matching
	// appRevisionPattern.
	HasAppRevisionArg bool

	// RunImageSpecPath is used by the split container buildpack
	RunImageSpecPath string
}

// NewDockerfileBuildpack creates a Dockerfile buildpack
func NewDockerfileBuildpack() *DockerfileBuildpack {
	return &DockerfileBuildpack{}
}

var successfulBuildRE = regexp.MustCompile(`Successfully built (\w+)`)

// Build implements Buildpack.Build
func (d *DockerfileBuildpack) Build(c *sous.BuildContext, dr *sous.DetectResult) (*sous.BuildResult, error) {
	start := time.Now()
	offset := c.Source.OffsetDir
	if offset == "" {
		offset = "."
	}

	cmd := []interface{}{"build"}
	r := dr.Data.(detectData)
	if r.HasAppVersionArg {
		v := c.Version().Version
		v.Meta = ""
		cmd = append(cmd, "--build-arg", fmt.Sprintf("%s=%s", AppVersionBuildArg, v))
	}
	if r.HasAppRevisionArg {
		cmd = append(cmd, "--build-arg", fmt.Sprintf("%s=%s", AppRevisionBuildArg, c.Version().RevID()))
	}

	cmd = append(cmd, offset)

	output, err := c.Sh.Stdout("docker", cmd...)
	if err != nil {
		return nil, err
	}

	match := successfulBuildRE.FindStringSubmatch(string(output))
	if match == nil {
		return nil, fmt.Errorf("Couldn't find container id in:\n%s", output)
	}

	return &sous.BuildResult{
		ImageID:    match[1],
		Elapsed:    time.Since(start),
		Advisories: c.Advisories,
	}, nil
}

// Detect detects if c has a Dockerfile or not.
func (d *DockerfileBuildpack) Detect(c *sous.BuildContext) (*sous.DetectResult, error) {
	dfPath := filepath.Join(c.Source.OffsetDir, "Dockerfile")
	if !c.Sh.Exists(dfPath) {
		return nil, fmt.Errorf("%s does not exist", dfPath)
	}
	sh := c.Sh.Clone()
	sh.LongRunning(false)
	df, err := sh.Stdout("cat", dfPath)
	if err != nil {
		return nil, err
	}
	hasAppVersion := appVersionPattern.MatchString(df)
	hasAppRevision := appRevisionPattern.MatchString(df)
	sous.Log.Debug.Printf("Detected a dockerfile at %q. Accepts version: %t, accepts revision: %t", dfPath, hasAppVersion, hasAppRevision)
	result := &sous.DetectResult{Compatible: true, Data: detectData{
		HasAppVersionArg:  hasAppVersion,
		HasAppRevisionArg: hasAppRevision,
	}}
	return result, nil
}
