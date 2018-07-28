package smoke

import (
	"fmt"
	"sort"
	"strings"
)

type matrixDef struct {
	OrderedDimensionNames []string
	OrderedDimensionDescs []string
	Dimensions            map[string]map[string]interface{}
}

type combination []particle

type particle struct {
	Dimension, Name string
	Value           interface{}
}

func newMatrix() matrixDef {
	return matrixDef{Dimensions: map[string]map[string]interface{}{}}
}

func (m matrixDef) PrintDimensions() {
	var out []string
	for _, name := range m.OrderedDimensionNames {
		out = append(out, "<"+name+">")
	}
	fmt.Printf("Matrix dimensions: <top-level>/%s\n", strings.Join(out, "/"))
	for i, name := range m.OrderedDimensionNames {
		desc := m.OrderedDimensionDescs[i]
		fmt.Printf("Dimension %s: %s (", name, desc)
		d := m.Dimensions[name]
		for valueName := range d {
			fmt.Printf(" %s", valueName)
		}
		fmt.Print(" )\n")
	}
}

// AddDimension adds a new dimension to this matrix with the provided name
// and desc which is used in help text when using -matrix flag on 'go test'.
// The values are a map of short value names to concrete representations, which
// are passed to tests. The names of values map to parts of the sub-test path
// for 'go test -run' flag.
func (m *matrixDef) AddDimension(name, desc string, values map[string]interface{}) {
	m.OrderedDimensionNames = append(m.OrderedDimensionNames, name)
	m.OrderedDimensionDescs = append(m.OrderedDimensionDescs, desc)
	m.Dimensions[name] = values
}

// FixedDimension returns a matrixDef based on m with one of its dimensions
// fixed to a particular value. This can be used when writing tests where
// only a single value for one particular dimension is appropriate.
func (m matrixDef) FixedDimension(dimensionName, valueName string) matrixDef {
	return m.clone(func(dimension, value string) bool {
		return dimension != dimensionName || value == valueName
	})
}

func (m matrixDef) clone(include func(dimension, value string) bool) matrixDef {
	n := m
	n.Dimensions = map[string]map[string]interface{}{}
	for name, values := range m.Dimensions {
		nv := map[string]interface{}{}
		for vn, v := range values {
			if include(name, vn) {
				nv[vn] = v
			}
		}
		n.Dimensions[name] = nv
	}
	return n
}

func (m *matrixDef) combinations() []combination {
	combos := [][]combination{}
	for _, d := range m.OrderedDimensionNames {
		c := []combination{}
		dim := m.Dimensions[d]
		valNames := []string{}
		for name := range dim {
			valNames = append(valNames, name)
		}
		sort.Strings(valNames)
		for _, name := range valNames {
			c = append(c, combination{
				particle{
					Dimension: d,
					Name:      name,
					Value:     dim[name],
				},
			})
		}
		combos = append(combos, c)
	}
	return product(combos...)
}

func product(slices ...[]combination) []combination {
	res := slices[0]
	for _, s := range slices[1:] {
		res = mult(res, s)
	}
	return res
}

func mult(a, b []combination) []combination {
	res := make([][]combination, len(a)*len(b))
	n := 0
	for _, aa := range a {
		for _, bb := range b {
			res[n] = []combination{aa, bb}
			n++
		}
	}
	slice := make([]combination, len(res))
	for i, r := range res {
		slice[i] = concat(r)
	}
	return slice
}

func concat(combos []combination) combination {
	res := combos[0]
	for _, c := range combos[1:] {
		res = append(res, c...)
	}
	return res
}

// String returns the sub-test path of this combination. E.g.
// dim1ValueName/dim2ValueName[/...].
func (c combination) String() string {
	var names []string
	for _, p := range c {
		names = append(names, p.Name)
	}
	return strings.Join(names, "/")
}

// Map returns a map of dimension name to specific value for this combination.
// This is useful where we want to look up a value by dimension name.
func (c combination) Map() map[string]interface{} {
	res := make(map[string]interface{}, len(c))
	for _, p := range c {
		res[p.Dimension] = p.Value
	}
	return res
}
