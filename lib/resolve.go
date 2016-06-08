package sous

import (
	"log"
	"strings"

	"github.com/opentable/sous/util/hy"
	"github.com/opentable/sous/util/yaml"
)

// MissingImageNamesError reports that we couldn't get names for one or more source versions
type MissingImageNamesError struct {
	Causes []error
}

// Resolve drives the Sous deployment resolution process. It calls out to the
// appropriate components to compute the intended deployment set, collect the
// actual set, compute the diffs and then issue the commands to rectify those
// differences.
func Resolve(nc NameCache, config State) error {
	gdm, err := config.Deployments()
	if err != nil {
		return err
	}

	rc := NewRectiAgent(nc)

	err = guardImageNamesKnown(rc, gdm)
	if err != nil {
		return err
	}

	ads, err := GetRunningDeploymentSet(baseURLs(config))
	if err != nil {
		return err
	}

	differ := ads.Diff(gdm)

	errs := make(chan RectificationError)

	Rectify(differ, errs, rc)

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

//ResolveFromDir does everything that Resolve does, plus it adds loading the
//Sous config from a directory of YAML files. This use case is important for
//proof-of-concept, but long term we expect to be able to abstract the storage
//of the Sous state away, so this might be deprecated at some point.
func ResolveFromDir(nc NameCache, dir string) error {
	config, err := loadConfig(dir)
	if err != nil {
		return err
	}

	return Resolve(nc, config)
}

func loadConfig(dir string) (st State, err error) {
	u := hy.NewUnmarshaler(yaml.Unmarshal)
	err = u.Unmarshal(dir, &st)
	return
}

func baseURLs(st State) []string {
	urls := make([]string, 0, len(st.Defs.Clusters))
	for _, cl := range st.Defs.Clusters {
		urls = append(urls, cl.BaseURL)
	}
	return urls
}
