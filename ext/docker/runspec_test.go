package docker

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestRunspecLoadLegacyManifest(t *testing.T) {

	mBuf := bytes.NewBufferString(`{
  "image": {
    "type": "Docker",
    "from": "scratch"
  },
  "files": [
    {
      "source": { "dir": "/built"},
      "dest":   { "dir": "/"}
    }
  ],
  "exec": ["/sous-demo"]
	}`)

	runspec := &MultiImageRunSpec{}
	dec := json.NewDecoder(mBuf)
	dec.Decode(runspec)

	if runspec.Image.From != "scratch" {
		t.Error("RunSpec didn't load Image.From")
	}

	if len(runspec.Files) != 1 {
		t.Fatal("No files loaded")
	}
	if runspec.Files[0].Source.Dir != "/built" {
		t.Error("RunSpec didn't load Files[0].Source")
	}
	if runspec.Files[0].Destination.Dir != "/" {
		t.Error("RunSpec didn't load Files[0].Destination")
	}

	if len(runspec.Validate()) > 0 {
		t.Error("Expected RunSpec to validate")
	}

	nrs := runspec.Normalized()
	if len(nrs.Images) != 1 {
		t.Error("Normalized runspec doesn't have 1 Images [sic]")
	}
}

func TestRunspecLoadMultiManifest(t *testing.T) {

	mBuf := bytes.NewBufferString(`{
		"images": [
			{
				"image": {
					"type": "Docker",
					"from": "scratch"
				},
				"files": [
					{
						"source": { "dir": "/built"},
						"dest":   { "dir": "/"}
					}
				],
				"exec": ["/sous-demo"]
		  },
			{
				"image": {
					"type": "Docker",
					"from": "scratch"
				},
				"files": [
					{
						"source": { "dir": "/built-extra"},
						"dest":   { "dir": "/"}
					}
				],
				"exec": ["/sous-scratch"]
		  }
    ]
	}`)

	runspec := &MultiImageRunSpec{}
	dec := json.NewDecoder(mBuf)
	err := dec.Decode(runspec)
	if err != nil {
		t.Fatal(err)
	}

	if len(runspec.Images) != 2 {
		t.Fatal("runspec doesn't have 2 Images [sic]")
	}

	if runspec.Images[0].Image.From != "scratch" {
		t.Error("RunSpec didn't load Image.From")
	}

	if len(runspec.Images[0].Files) != 1 {
		t.Fatal("No files loaded")
	}
	if runspec.Images[0].Files[0].Source.Dir != "/built" {
		t.Error("RunSpec didn't load Files[0].Source")
	}
	if runspec.Images[0].Files[0].Destination.Dir != "/" {
		t.Error("RunSpec didn't load Files[0].Destination")
	}

	if len(runspec.Validate()) > 0 {
		t.Error("Expected RunSpec to validate")
	}

	nrs := runspec.Normalized()
	if len(nrs.Images) != 2 {
		t.Error("Normalized runspec doesn't have 2 Images [sic]")
	}

	runspec.SplitImageRunSpec = &SplitImageRunSpec{
		Image: sbmImage{From: "scratch"},
	}

	if len(runspec.Validate()) == 0 {
		t.Error("Expected RunSpec not to validate with mixed legacy/new data")
	}

}
func TestRunSpecValidate(t *testing.T) {
	rs := &SplitImageRunSpec{}
	flaws := rs.Validate()
	if len(flaws) != 4 {
		t.Errorf("Expected %d flaws, got %d", 4, len(flaws))
	}
}
