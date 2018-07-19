package smoke

type sousFlags struct {
	kind    string
	flavor  string
	cluster string
	repo    string
	offset  string
	tag     string
}

// ManifestIDFlags returns a derived set of flags only keeping those that play a
// part in identifying a manifest.
func (f *sousFlags) ManifestIDFlags() *sousFlags {
	if f == nil {
		return nil
	}
	return &sousFlags{
		repo:   f.repo,
		offset: f.offset,
		flavor: f.flavor,
	}
}

// ManifestIDFlags returns a derived set of flags only keeping those that play a
// part in identifying a deployment.
func (f *sousFlags) DeploymentIDFlags() *sousFlags {
	if f == nil {
		return nil
	}
	didFlags := f.ManifestIDFlags()
	didFlags.cluster = f.cluster
	return didFlags
}

func (f *sousFlags) SousDeployFlags() *sousFlags {
	if f == nil {
		return nil
	}
	deployFlags := f.DeploymentIDFlags()
	deployFlags.tag = f.tag
	return deployFlags
}

// SousInitFlags returns a derived set of flags only keeping those that play a
// part in the 'sous init' command.
func (f *sousFlags) SousInitFlags() *sousFlags {
	if f == nil {
		return nil
	}
	initFlags := f.ManifestIDFlags()
	initFlags.kind = f.kind
	return initFlags
}

func (f *sousFlags) SourceIDFlags() *sousFlags {
	if f == nil {
		return nil
	}
	sidFlags := f.ManifestIDFlags()
	sidFlags.tag = f.tag
	return sidFlags
}

// Map returns a map of flag names to values.
// Any flags which have an empty string value are omitted from the map entirely.
func (f *sousFlags) FlagMap() map[string]string {
	m := map[string]string{}
	if f == nil {
		return m
	}
	if f.kind != "" {
		m["kind"] = f.kind
	}
	if f.flavor != "" {
		m["flavor"] = f.flavor
	}
	if f.cluster != "" {
		m["cluster"] = f.cluster
	}
	if f.repo != "" {
		m["repo"] = f.repo
	}
	if f.offset != "" {
		m["offset"] = f.offset
	}
	if f.tag != "" {
		m["tag"] = f.tag
	}
	return m
}

func (f *sousFlags) FlagPrefix() string {
	return "-"
}
