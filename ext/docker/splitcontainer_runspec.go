package docker

import (
	"strings"

	sous "github.com/opentable/sous/lib"
)

// A SplitImageRunSpec is the JSON structure that describes an individual deploy container.
type SplitImageRunSpec struct {
	// Kind is used to denote that the image isn't a "normal" i.e. deployable
	// service image, but instead some other kind. Examples include "builder" or
	// "test" or "uploader"
	Kind string `json:"kind"`

	// Offset is the usual Sous offset: the path relative to the root of the
	// project that selects one of several services built from the same codebase.
	Offset string `json:"offset"`

	// Image describes the base of the deploy image. type should be "docker" and
	// from should be a suitable FROM image.
	Image sbmImage `json:"image"`

	// Files maps directories of built files in the build container to their
	// destination in the deploy container.
	Files []sbmInstall `json:"files"`

	// Exec describes the command to ultimately run in the deploy container -
	// essentially a Docker ENTRYPOINT
	Exec []string `json:"exec"`
}

// A MultiImageRunSpec is the JSON structure that build containers emit
// in order that their associated deploy containers can be assembled
// It *can* parse the same structure as SplitImageRunSpec, because there are a
// few builds that already use that format.
type MultiImageRunSpec struct {
	*SplitImageRunSpec `json:",omitempty"`
	Images             []SplitImageRunSpec `json:"images"`
}

type sbmImage struct {
	Type string `json:"type"`
	From string `json:"from"`
}

type sbmInstall struct {
	Source      sbmFile `json:"source"`
	Destination sbmFile `json:"dest"`
}

type sbmFile struct {
	Dir string `json: "dir"`
}

// Validate implements Flawed on MultiImageRunSpec
func (ms *MultiImageRunSpec) Validate() []sous.Flaw {
	fs := []sous.Flaw{}
	if ms.SplitImageRunSpec != nil {
		if len(ms.Images) > 0 {
			fs = append(fs, sous.FatalFlaw("Uses both legacy fields and list of images!"))
		}
		fs = append(fs, ms.SplitImageRunSpec.Validate()...)
	} else {
		for idx, spec := range ms.Images {
			sfs := spec.Validate()
			for _, f := range sfs {
				f.AddContext("image %d", idx)
			}
			fs = append(fs, sfs...)
		}
	}
	return fs
}

// Normalized returns a MultiImageRunSpec where all the SplitImageRunSpecs are in Images.
func (ms MultiImageRunSpec) Normalized() MultiImageRunSpec {
	if ms.SplitImageRunSpec == nil {
		return ms
	}
	return MultiImageRunSpec{
		Images: []SplitImageRunSpec{*ms.SplitImageRunSpec},
	}
}

// Validate implements Flawed on SplitImageRunSpec
func (rs *SplitImageRunSpec) Validate() []sous.Flaw {
	fs := []sous.Flaw{}
	if strings.ToLower(rs.Image.Type) != "docker" {
		fs = append(fs, sous.FatalFlaw("Only 'docker' is recognized currently as an image type, was %q", rs.Image.Type))
	}
	if rs.Image.From == "" {
		fs = append(fs, sous.FatalFlaw("Required image.from was empty or missing."))
	}
	if len(rs.Files) == 0 {
		fs = append(fs, sous.FatalFlaw("Deploy image doesn't make sense with empty list of files."))
	}
	if len(rs.Exec) == 0 {
		fs = append(fs, sous.FatalFlaw("Need an exec list."))
	}

	return fs
}
