package docker

import (
	"regexp"
	"strings"

	"github.com/opentable/sous/lib"
	"github.com/pkg/errors"
	"github.com/samsalisbury/semv"
)

// SourceIDFromLabels builds a SourceID from a map of labels, generally
// acquired from a Docker image.
func SourceIDFromLabels(labels map[string]string) (sous.SourceID, error) {
	missingLabels := make([]string, 0, 3)
	repo, present := labels[DockerRepoLabel]
	if !present {
		missingLabels = append(missingLabels, DockerRepoLabel)
	}

	versionStr, present := labels[DockerVersionLabel]
	if !present {
		missingLabels = append(missingLabels, DockerVersionLabel)
	}

	revision, present := labels[DockerRevisionLabel]
	if !present {
		missingLabels = append(missingLabels, DockerRevisionLabel)
	}

	path, present := labels[DockerPathLabel]
	if !present {
		missingLabels = append(missingLabels, DockerPathLabel)
	}

	if len(missingLabels) > 0 {
		err := errors.Errorf("Missing labels: %v", missingLabels)
		return sous.SourceID{}, err
	}

	version, err := semv.Parse(versionStr)
	version.Meta = revision

	return sous.SourceID{
		Repo:    repo,
		Version: version,
		Dir:     path,
	}, err
}

var stripRE = regexp.MustCompile("^([[:alpha:]]+://)?(github.com(/opentable)?)?")

// Labels computes a map of labels that should be applied to a container
// image that is built based on this SourceID.
func Labels(sid sous.SourceID) map[string]string {
	labels := make(map[string]string)
	labels[DockerVersionLabel] = sid.Version.Format(`M.m.p-?`)
	labels[DockerRevisionLabel] = sid.RevID()
	labels[DockerPathLabel] = string(sid.Dir)
	labels[DockerRepoLabel] = string(sid.Repo)
	return labels
}

func imageNameBase(sid sous.SourceID) string {
	name := string(sid.Repo)

	name = stripRE.ReplaceAllString(name, "")
	if string(sid.Dir) != "" {
		name = strings.Join([]string{name, string(sid.Dir)}, "/")
	}
	return name
}

func tagName(v semv.Version) string {
	return v.Format("M.m.p-?")
}

func versionName(sid sous.SourceID) string {
	return strings.Join([]string{imageNameBase(sid), tagName(sid.Version)}, ":")
}

func revisionName(sid sous.SourceID) string {
	return strings.Join([]string{imageNameBase(sid), sid.RevID()}, ":")
}
