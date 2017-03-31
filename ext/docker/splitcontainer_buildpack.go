package docker

import (
	"log"
	"os"
	"path/filepath"

	"github.com/docker/docker/builder/dockerfile/parser"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/pkg/errors"
)

type (
	// A SplitBuildpack implements the pattern of using a build container and producing a separate deploy container
	SplitBuildpack struct {
		registry docker_registry.Client
	}
)

const SOUS_BUILD_MANIFEST = "SOUS_BUILD_MANIFEST"

// NewSplitBuildpack returns a new SplitBuildpack
func NewSplitBuildpack(r docker_registry.Client) *SplitBuildpack {
	return &SplitBuildpack{
		registry: r,
	}
}

func parseDockerfile(path string) (*parser.Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	d := parser.Directive{LookingForDirectives: true}
	parser.SetEscapeToken(parser.DefaultEscapeToken, &d)

	return parser.Parse(f, &d)
}

// Detect implements Buildpack on SplitBuildpack
func (sbp *SplitBuildpack) Detect(ctx *sous.BuildContext) (*sous.DetectResult, error) {
	dfPath := filepath.Join(ctx.Source.OffsetDir, "Dockerfile")
	if !ctx.Sh.Exists(dfPath) {
		return nil, errors.Errorf("%s does not exist", dfPath)
	}

	froms := []*parser.Node{}
	envs := []*parser.Node{}
	ast, err := parseDockerfile(ctx.Sh.Abs(dfPath))
	if err != nil {
		return nil, err
	}

	// Parse for ENV SOUS_BUILD_MANIFEST
	// Parse for FROM
	for n, node := range ast.Children {
		log.Printf("%d %#v", n, node)
		switch node.Value {
		case "env":
			envs = append(envs, node.Next)
			log.Printf("%d %#v", n, node.Next)
		case "from":
			froms = append(froms, node.Next)
			log.Printf("%d %#v", n, node.Next)
		}
	}

	for _, e := range envs {
		if e.Value == SOUS_BUILD_MANIFEST {
			return &sous.DetectResult{Compatible: true}, nil
		}
	}

	for _, f := range froms {
		md, err := sbp.registry.GetImageMetadata(f.Value, "")
		if err != nil {
			continue
		}
		if _, ok := md.Env[SOUS_BUILD_MANIFEST]; ok {
			return &sous.DetectResult{Compatible: true}, nil
		}
	}
	//   present? -> true
	// Fetch from-manifest
	// Inspect that for ENV SOUS_BUILD_MANIFEST
	//   return present?
	return &sous.DetectResult{Compatible: false}, nil
}

// Build implements Buildpack on SplitBuildpack
func (sbp *SplitBuildpack) Build(ctx *sous.BuildContext, drez *sous.DetectResult) (*sous.BuildResult, error) {
	return nil, nil
}
