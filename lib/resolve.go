package sous

import "strings"

type (
	// Resolver is responsible for resolving intended and actual deployment
	// states.
	Resolver struct {
		Deployer      Deployer
		Registry      Registry
		IntendedState State
	}
	// ResolveErrors collect all the errors for a resolve action into a single
	// error to be handled elsewhere
	ResolveErrors struct {
		Causes []error
	}

	// MissingImageNamesError reports that we couldn't get names for one or more
	// source versions
	MissingImageNamesError struct {
		Causes []error
	}
)

func NewResolver(d Deployer, r Registry, intended State) *Resolver {
	return &Resolver{
		Deployer:      d,
		Registry:      r,
		IntendedState: intended,
	}
}

func (re *ResolveErrors) Error() string {
	s := []string{"Errors during resolve:"}
	for _, e := range re.Causes {
		s = append(s, e.Error())
	}
	return strings.Join(s, "\n  ")
}

// Resolve drives the Sous deployment resolution process. It calls out to the
// appropriate components to compute the intended deployment set, collect the
// actual set, compute the diffs and then issue the commands to rectify those
// differences.
func (r *Resolver) Resolve() error {
	return r.ResolveFilteredDeployments(nil)
}

// ResolveFilteredDeployments is similar to Resolve, but also accepts a
// predicate to filter those deployments. See Deploments.Filter for details.
func (r *Resolver) ResolveFilteredDeployments(pr DeploymentPredicate) error {
	Log.Debug.Print("Loading GDM")
	gdm, err := r.IntendedState.Deployments()
	gdm = gdm.Filter(pr)
	if err != nil {
		return err
	}

	Log.Debug.Print("Loaded. Collecting ADC...")

	ads, err := r.Deployer.GetRunningDeployment(r.IntendedState.BaseURLs())
	if err != nil {
		return err
	}

	Log.Debug.Print("Collected. Checking readiness to deploy...")

	if err := guardImageNamesKnown(r.Registry, gdm); err != nil {
		return err
	}

	Log.Debug.Print("Looks good. Proceeding...")

	differ := ads.Diff(gdm)

	errs := Rectify(differ, r.Deployer, r.Registry)

	re := &ResolveErrors{Causes: []error{}}
	for err := range errs {
		re.Causes = append(re.Causes, err)
		Log.Debug.Printf("resolve error = %+v\n", err)
	}

	if len(re.Causes) > 0 {
		return re
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

func guardImageNamesKnown(r Registry, gdm Deployments) error {
	es := make([]error, 0, len(gdm))
	for _, d := range gdm {
		_, err := r.GetArtifact(d.SourceVersion)
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
//func (r *Resolver) ResolveFromDir(dir string) error {
//	return r.ResolveFromDirFiltered(dir, nil)
//}
//
//// ResolveFromDirFiltered is similar to ResolveFromDir, but additionally filters
//// the deployments to be resolved based on the predicate.
//func (r *Resolver) ResolveFromDirFiltered(dir string, pr DeploymentPredicate) error {
//	config, err := LoadState(dir)
//	if err != nil {
//		return err
//	}
//	return r.ResolveFilteredDeployments(pr)
//}
