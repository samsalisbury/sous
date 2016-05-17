package sous

import (
	"fmt"

	"github.com/opentable/sous/util/docker_registry"
)

type (
	NameCache struct {
		DockerNameLookup
		SourceNameLookup
	}

	SourceRecord struct {
		SourceVersion SourceVersion
		etag          string
	}

	ImageName string

	SourceNameLookup map[ImageName]SourceRecord
	DockerNameLookup map[SourceVersion]ImageName

	NotModifiedErr struct{}

	NoImageNameFound struct {
		SourceVersion
	}

	NoSourceVersionFound struct {
		ImageName
	}
)

var theNameCache = NameCache{
	make(DockerNameLookup),
	make(SourceNameLookup),
}

func (e NoImageNameFound) Error() string {
	return fmt.Sprintf("No image name for %v", e.SourceVersion)
}

func (e NoSourceVersionFound) Error() string {
	return fmt.Sprintf("No image name for %v", e.ImageName)
}

func (e NotModifiedErr) Error() string {
	return "Not modified"

}

func (dl DockerNameLookup) GetImageName(sv SourceVersion) (ImageName, error) {
	if in, ok := dl[sv]; ok {
		return in, nil
	} else {
		return "", NoImageNameFound{sv}
	}
}

func (sn SourceNameLookup) getSourceRecord(in ImageName) (SourceRecord, error) {
	if sr, ok := sn[in]; ok {
		return sr, nil
	} else {
		return SourceRecord{}, NoSourceVersionFound{in}
	}
}

func (sn SourceNameLookup) GetSourceVersion(in ImageName) (SourceVersion, error) {
	if sr, ok := sn[in]; ok {
		return sr.SourceVersion, nil
	} else {
		return SourceVersion{}, NoSourceVersionFound{in}
	}
}

func (dl DockerNameLookup) InsertDockerName(sv SourceVersion, in ImageName) error {
	dl[sv] = in
	return nil
}

func (sn SourceNameLookup) InsertSourceVersion(in ImageName, sv SourceVersion, etag string) error {
	sn[in] = SourceRecord{sv, etag}
	return nil
}

func (nc *NameCache) Insert(sv SourceVersion, in ImageName, etag string) error {
	err := nc.InsertSourceVersion(in, sv, etag)
	if err != nil {
		return err
	}

	err = nc.InsertDockerName(sv, in)
	if err != nil {
		return err
	}

	return nil
}

// GetSourceVersion retreives a source version for an image name, updating it from the server if necessary
// Each call to GetSourceVersion implies an HTTP request, although it may be abbreviated by the use of an etag.
func GetSourceVersion(dr docker_registry.Client, in ImageName) (SourceVersion, error) {
	sr, err := theNameCache.getSourceRecord(in)

	newSV, etag, err := retreiveSourceVersion(dr, in, sr.etag)
	if _, ok := err.(NotModifiedErr); ok {
		return sr.SourceVersion, nil
	}
	if err != nil {
		return SourceVersion{}, err
	}

	if sr.SourceVersion != newSV {
		theNameCache.Insert(newSV, in, etag)
	}
	return newSV, nil
}

func retreiveSourceVersion(dc docker_registry.Client, in ImageName, etag string) (SourceVersion, string, error) {
	md, err := dc.GetImageMetadata(string(in), etag)
	if err != nil {
		return SourceVersion{}, "", err
	}

	sv, err := SourceVersionFromLabels(md.Labels)
	if err != nil {
		return SourceVersion{}, "", err
	}

	return sv, md.Etag, err
}
