package sous

import "log"

func Resolve(dir string) error {
	config, err := loadConfig(dir)
	if err != nil {
		return err
	}

	gdm := collectIntendedSet(config)
	ads,err := collectActualSet(baseURLs(config))
	if err != nil {
		return err
	}

	differ := buildDiffer(gdm, ads)
	errs, done := Rectify(differ, rc)
	go handleErrors(errs)
	<-done
	return nil
}

func handleErrors(errs chan RectificationError) {
	go func() {
		for err := range errs {
			log.Printf("err = %+v\n", err)
		}
	}()
}

func loadConfig(dir string) (st State, err error) {
	u := hy.NewUnmarshaler(yaml.Unmarshal)
	err = u.Unmarshal(dir, &st)
	return
}

func baseURLs(st State) []string {
	urls := make([]string, len(st.Defs.Clusters))
	for cl := range st.Defs.Clusters {
		urls = append(urls, cl.BaseURL)
	}
	return urls
}

func collectIntendedSet(st State) Deployments {
	deps := make(Deployments)
	for m := range st.Manifests {
		for ds := m.Deployments {
			deps = append(deps, BuildDeployment(m, ds))
		}
	}
	return deps
}

func collectActualSet(singularityUrls []string) (Deployments, error) {
	return GetRunningDeploymentSet(singularityUrls)
}

func buildDiffer(gdm, ads Deployments) DiffChans {
	return gdm.Diff(ads)
}
