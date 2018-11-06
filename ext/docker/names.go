package docker

import (
	"crypto/sha1"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
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
func Labels(sid sous.SourceID, rev string) map[string]string {
	labels := make(map[string]string)
	labels[DockerVersionLabel] = sid.Version.Format(`M.m.p-?`)
	labels[DockerRevisionLabel] = rev
	labels[DockerPathLabel] = sid.Location.Dir
	labels[DockerRepoLabel] = sid.Location.Repo
	return labels
}

// XXX The idea is to make this a configuration value at some point.
var stripRE = regexp.MustCompile("^([[:alpha:]]+://)?(github.com(/opentable(/)?)?)?")

func imageRepoName(sl sous.SourceLocation, kind string) string {
	var name string
	if strings.HasPrefix(sl.Repo, "github.com") {
		name = imageRepoNameGitHub(sl.Repo, sl.Dir)
	} else {
		name = imageRepoNameGeneric(sl.Repo, sl.Dir)
	}

	if kind == "" {
		return name
	}

	return name + "-" + kind
}

func imageRepoNameGeneric(repo, offset string) string {
	name := repo
	if offset != "" {
		name += "/" + offset
	}
	return fmt.Sprintf("%x", sha1.Sum([]byte(name)))
}

func imageRepoNameGitHub(repo, dir string) string {
	name := stripRE.ReplaceAllString(repo, "")
	if dir != "" {
		name = name + "/" + dir
	}
	return strings.ToLower(name)
}

func tagName(v semv.Version) string {
	return v.Format("M.m.p-?")
}

func versionName(sid sous.SourceID, kind string) string {
	return strings.Join([]string{imageRepoName(sid.Location, kind), tagName(sid.Version)}, ":")
}

func revisionName(sid sous.SourceID, rev string, kind string, time time.Time) string {
	//A tag name must be valid ASCII and may contain lowercase and uppercase
	//letters, digits, underscores, periods and dashes. A tag name may not start
	//with a period or a dash and may contain a maximum of 128 characters.
	//
	// revID = 40 bytes
	// RFC3339(ish) timestamp = 26 bytes
	// 40 + 26 + 2(separators) = 68 < 128

	// z prefix sorts "pinning" labels to the bottom
	// Format is the RFC3339 format, with . instead of : so that it's a legal docker tag
	labelStr := fmt.Sprintf("z%v-%v", rev, time.UTC().Format("2006-01-02T15.04.05"))
	return strings.Join([]string{imageRepoName(sid.Location, kind), labelStr}, ":")
}

func fullRepoName(registryHost string, sl sous.SourceLocation, kind string, ls logging.LogSink) string {
	frn := filepath.Join(registryHost, imageRepoName(sl, kind))
	messages.ReportLogFieldsMessage("Repo name", logging.DebugLevel, ls, sl, logging.KV("full-rep-name", frn))
	return frn
}

func versionTag(registryHost string, v sous.SourceID, kind string, ls logging.LogSink) string {
	verTag := filepath.Join(registryHost, versionName(v, kind))
	messages.ReportLogFieldsMessage("Docker Version Tag", logging.DebugLevel, ls, kind, logging.KV("version-tag", v))
	return verTag
}

func revisionTag(registryHost string, v sous.SourceID, rev string, kind string, time time.Time, ls logging.LogSink) string {
	revTag := filepath.Join(registryHost, revisionName(v, rev, kind, time))
	messages.ReportLogFieldsMessage("Docker RevisionTag", logging.DebugLevel, ls, kind, time, logging.KV("revision-tag", revTag), v)
	return revTag
}
