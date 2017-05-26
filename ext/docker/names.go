package docker

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

	id, err := sous.NewSourceID(repo, path, versionStr)
	id.Version.Meta = revision
	return id, err
}

// Labels computes a map of labels that should be applied to a container
// image that is built based on this SourceID.
func Labels(sid sous.SourceID) map[string]string {
	labels := make(map[string]string)
	labels[DockerVersionLabel] = sid.Version.Format(`M.m.p-?`)
	labels[DockerRevisionLabel] = sid.RevID()
	labels[DockerPathLabel] = sid.Location.Dir
	labels[DockerRepoLabel] = sid.Location.Repo
	return labels
}

var stripRE = regexp.MustCompile("^([[:alpha:]]+://)?(github.com(/opentable)?)?")

func imageRepoName(sl sous.SourceLocation, kind string) string {
	name := sl.Repo

	name = stripRE.ReplaceAllString(name, "")
	if sl.Dir != "" {
		name = strings.Join([]string{name, sl.Dir}, "/")
	}

	if kind == "" {
		return name
	}

	return strings.Join([]string{name, kind}, "-")
}

func tagName(v semv.Version) string {
	return v.Format("M.m.p-?")
}

func versionName(sid sous.SourceID, kind string) string {
	return strings.Join([]string{imageRepoName(sid.Location, kind), tagName(sid.Version)}, ":")
}

func revisionName(sid sous.SourceID, kind string, time time.Time) string {
	//A tag name must be valid ASCII and may contain lowercase and uppercase
	//letters, digits, underscores, periods and dashes. A tag name may not start
	//with a period or a dash and may contain a maximum of 128 characters.
	//
	// revID = 40 bytes
	// RFC3339(ish) timestamp = 26 bytes
	// 40 + 26 + 2(separators) = 68 < 128

	// z prefix sorts "pinning" labels to the bottom
	// Format is the RFC3339 format, with . instead of : so that it's a legal docker tag
	labelStr := fmt.Sprintf("z%v-%v", sid.RevID(), time.Format("2006-01-02T15.04.05Z07.00"))
	return strings.Join([]string{imageRepoName(sid.Location, kind), labelStr}, ":")
}

func fullRepoName(registryHost string, sl sous.SourceLocation, kind string) string {
	frn := filepath.Join(registryHost, imageRepoName(sl, kind))
	Log.Debug.Printf("Repo name: % #v => %q", sl, frn)
	return frn
}

func versionTag(registryHost string, v sous.SourceID, kind string) string {
	verTag := filepath.Join(registryHost, versionName(v, kind))
	Log.Debug.Printf("Version tag: % #v => %s", v, verTag)
	return verTag
}

func revisionTag(registryHost string, v sous.SourceID, kind string, time time.Time) string {
	revTag := filepath.Join(registryHost, revisionName(v, kind, time))
	Log.Debug.Printf("RevisionTag: % #v => %s", v, revTag)
	return revTag
}
