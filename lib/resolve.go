package sous

import (
	"log"

	"github.com/opentable/sous/util/hy"
	"github.com/opentable/sous/util/yaml"
)

// Resolve drives the Sous deployment resolution process. It calls out to the
// appropriate components to compute the intended deployment set, collect the
// actual set, compute the diffs and then issue the commands to rectify those
// differences.
func Resolve(nc NameCache, config State) error {
	gdm, err := config.Deployments()
	if err != nil {
		return err
	}

	ads, err := GetRunningDeploymentSet(baseURLs(config))
	if err != nil {
		return err
	}

	differ := ads.Diff(gdm)

	errs := make(chan RectificationError)

	rc := NewRectiAgent(nc)

	Rectify(differ, errs, rc)

	for err := range errs {
		log.Printf("err = %+v\n", err)
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
	log.Print(urls)
	return urls
}
