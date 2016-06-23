package sous

import (
	"log"
	"strings"
)

// MissingImageNamesError reports that we couldn't get names for one or more source versions
type MissingImageNamesError struct {
	Causes []error
}

// Resolve drives the Sous deployment resolution process. It calls out to the
// appropriate components to compute the intended deployment set, collect the
// actual set, compute the diffs and then issue the commands to rectify those
// differences.
func Resolve(rc RectificationClient, state State) error {
	return ResolveFilteredDeployments(rc, state, nil)
}

// ResolveFilteredDeployments is similar to Resolve, but also accepts a
// predicate to filter those deployments. See Deploments.Filter for details.
func ResolveFilteredDeployments(rc RectificationClient, state State, pr DeploymentPredicate) error {
	gdm, err := state.Deployments()
	gdm = gdm.Filter(pr)
	if err != nil {
		return err
	}

	err = guardImageNamesKnown(rc, gdm)
	if err != nil {
		return err
	}

	sc := NewSetCollector(rc)
	ads, err := sc.GetRunningDeployment(state.BaseURLs())
	if err != nil {
		return err
	}

	differ := ads.Diff(gdm)

	errs := Rectify(differ, rc)

	for err := range errs {
		log.Printf("err = %+v\n", err)
	}
	return nil
}

func (e *MissingImageNamesError) Error() string {
	causeStrs := make([]string, 0, len(e.Causes)+1)
	causeStrs = append(causeStrs, "Image names are unknown to Sous for source versions")
	for _, c := range e.Causes {
		causeStrs = append(causeStrs, c.Error())
	}
	return strings.Join(causeStrs, "  \n")
}

func guardImageNamesKnown(rc RectificationClient, gdm Deployments) error {
	es := make([]error, 0, len(gdm))
	for _, d := range gdm {
		_, err := rc.ImageName(d)
		if err != nil {
			es = append(es, err)
		}
	}
	if len(es) > 0 {
		return &MissingImageNamesError{es}
	}
	return nil
}

// ResolveFromDir does everything that Resolve does, plus it adds loading the
// Sous config from a directory of YAML files. This use case is important for
// proof-of-concept, but long term we expect to be able to abstract the storage
// of the Sous state away, so this might be deprecated at some point.
func ResolveFromDir(rc RectificationClient, dir string) error {
	return ResolveFromDirFiltered(rc, dir, nil)
}

// ResolveFromDirFiltered is similar to ResolveFromDir, but additionally filters
// the deployments to be resolved based on the predicate.
func ResolveFromDirFiltered(rc RectificationClient, dir string, pr DeploymentPredicate) error {
	config, err := LoadState(dir)
	if err != nil {
		return err
	}

	return ResolveFilteredDeployments(rc, config, pr)
}
