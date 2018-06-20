package sous

import (
	"fmt"

	"github.com/opentable/sous/util/restful"
)

// EmptyReceiver implements Comparable on Defs
func (ds *Defs) EmptyReceiver() restful.Comparable {
	return &Defs{}
}

// VariancesFrom implements Comparable on Defs
func (ds *Defs) VariancesFrom(other restful.Comparable) restful.Variances {
	switch od := other.(type) {
	default:
		return restful.Variances{"not a Defs"}
	case *Defs:
		return restful.Variances(ds.Diff(od))
	}
}

// Diff reports differences between two Defs.
func (ds *Defs) Diff(o *Defs) []string {
	vs := []string{}

	if ds.DockerRepo != o.DockerRepo {
		vs = append(vs, "DockerRepo differs")
	}

	if len(ds.Clusters) != len(o.Clusters) {
		vs = append(vs, "Different number of clusters")
	} else {
		for cn, c := range ds.Clusters {
			oc, has := o.Clusters[cn]
			if !has {
				vs = append(vs, fmt.Sprintf("Doesn't have %s", cn))
				continue
			}
			vs = append(vs, prefixed("cluster "+cn, c.Diff(oc))...)
		}
	}

	vs = append(vs, prefixed("resource ", ds.Resources.Diff(o.Resources))...)
	vs = append(vs, prefixed("metadata ", ds.Metadata.Diff(o.Metadata))...)
	vs = append(vs, prefixed("envdefs ", ds.EnvVars.Diff(o.EnvVars))...)

	return vs
}

// Diff reports the differences.
func (evs EnvDefs) Diff(os EnvDefs) []string {
	if len(evs) != len(os) {
		return []string{"lengths differ"}
	}

	edmap := map[string]EnvDef{}
	for _, ed := range evs {
		edmap[ed.Name] = ed
	}

	vs := []string{}

	for _, ed := range os {
		o, has := edmap[ed.Name]
		if !has {
			vs = append(vs, "doesn't have "+ed.Name)
		}

		vs = append(vs, prefixed(ed.Name, o.Diff(ed))...)
	}

	return vs
}

// Diff reports the differences.
func (ed EnvDef) Diff(o EnvDef) []string {
	vs := []string{}
	if ed.Name != o.Name {
		vs = append(vs, "names differ")
	}
	if ed.Desc != o.Desc {
		vs = append(vs, "descs differ")
	}
	if ed.Scope != o.Scope {
		vs = append(vs, "scopes differ")
	}
	if ed.Type != o.Type {
		vs = append(vs, "types differ")
	}
	return vs
}

// Diff reports the differences.
func (fds FieldDefinitions) Diff(os FieldDefinitions) []string {
	if len(fds) != len(os) {
		return []string{"lengths differ"}
	}

	fdmap := map[string]FieldDefinition{}
	for _, res := range fds {
		fdmap[res.Name] = res
	}

	vs := []string{}
	for _, res := range os {
		o, has := fdmap[res.Name]
		if !has {
			vs = append(vs, "doesn't have "+res.Name)
		}

		vs = append(vs, prefixed(res.Name, o.Diff(res))...)
	}

	return vs
}

// Diff reports the differences.
func (fd FieldDefinition) Diff(o FieldDefinition) []string {
	ds := []string{}
	if fd.Name != o.Name {
		ds = append(ds, "names differ")
	}
	if fd.Optional != o.Optional {
		ds = append(ds, "optionality differs")
	}

	if fd.Default != o.Default {
		ds = append(ds, "defaults differ")
	}

	if fd.Type != o.Type {
		ds = append(ds, "types differ")
	}

	return ds
}

func prefixed(prefix string, in []string) []string {
	out := []string{}
	for _, v := range in {
		out = append(out, prefix+v)
	}
	return out
}

// Diff reports the differences between two Clusters.
func (c *Cluster) Diff(oc *Cluster) []string {
	vs := []string{}
	if c.Name != oc.Name {
		vs = append(vs, "names differ")
	}
	if c.Kind != oc.Kind {
		vs = append(vs, "names differ")
	}
	if c.BaseURL != oc.BaseURL {
		vs = append(vs, "names differ")
	}
	if len(c.Env) != len(oc.Env) {
		vs = append(vs, "Env map different sizes")
	} else {
		for n, v := range c.Env {
			ov, has := oc.Env[n]
			if !has {
				vs = append(vs, "missing Envvar: "+n)
			} else if v != ov {
				vs = append(vs, "values differ for "+n)
			}
		}
	}

	vs = append(vs, prefixed("startup ", c.Startup.diff(oc.Startup))...)

	if len(c.AllowedAdvisories) != len(oc.AllowedAdvisories) {
		vs = append(vs, "advisories whitelist length differs")
	} else {
		aamap := map[string]struct{}{}
		for _, a := range c.AllowedAdvisories {
			aamap[a] = struct{}{}
		}
		for _, a := range oc.AllowedAdvisories {
			if _, has := aamap[a]; !has {
				vs = append(vs, "missing advisory "+a)
			}
		}
	}

	return vs
}
