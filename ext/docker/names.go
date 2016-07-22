package docker

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/opentable/sous/lib"
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
		err := fmt.Errorf("Missing labels on manifest for %v", missingLabels)
		return sous.SourceID{}, err
	}

	version, err := semv.Parse(versionStr)
	version.Meta = revision

	return sous.SourceID{
		RepoURL:    sous.RepoURL(repo),
		Version:    version,
		RepoOffset: sous.RepoOffset(path),
	}, err
}

var stripRE = regexp.MustCompile("^([[:alpha:]]+://)?(github.com(/opentable)?)?")

// Labels computes a map of labels that should be applied to a container
// image that is built based on this SourceID.
func Labels(sv sous.SourceID) map[string]string {
	labels := make(map[string]string)
	labels[DockerVersionLabel] = sv.Version.Format(`M.m.p-?`)
	labels[DockerRevisionLabel] = sv.RevID()
	labels[DockerPathLabel] = string(sv.RepoOffset)
	labels[DockerRepoLabel] = string(sv.RepoURL)
	return labels
}

func imageNameBase(sv sous.SourceID) string {
	name := string(sv.RepoURL)

	name = stripRE.ReplaceAllString(name, "")
	if string(sv.RepoOffset) != "" {
		name = strings.Join([]string{name, string(sv.RepoOffset)}, "/")
	}
	return name
}

func tagName(v semv.Version) string {
	return v.Format("M.m.p-?")
}

func versionName(sv sous.SourceID) string {
	return strings.Join([]string{imageNameBase(sv), tagName(sv.Version)}, ":")
}

func revisionName(sv sous.SourceID) string {
	return strings.Join([]string{imageNameBase(sv), sv.RevID()}, ":")
}
