package smoke

import (
	"fmt"
	"sort"
	"strings"
)

// Matrix is a set of named dimensions and possible values, the Cartesian
// product of these values is used to produce Scenarios which are handed down
// to your tests.
type Matrix struct {
	orderedDimensionNames []string
	orderedDimensionDescs []string
	dimensions            map[string]map[string]interface{}
}

// Scenario is a single combination of values from a Matrix.
type Scenario []Binding

// Binding is a named value belonging to a particular dimension.
type Binding struct {
	Dimension, Name string
	Value           interface{}
}

// New returns a new Matrix.
func New() Matrix {
	return Matrix{dimensions: map[string]map[string]interface{}{}}
}

// PrintDimensions writes the dimensions an allowed values to stdout.
func (m Matrix) PrintDimensions() {
	var out []string
	for _, name := range m.orderedDimensionNames {
		out = append(out, "<"+name+">")
	}
	fmt.Printf("Matrix dimensions: <top-level>/%s\n", strings.Join(out, "/"))
	for i, name := range m.orderedDimensionNames {
		desc := m.orderedDimensionDescs[i]
		fmt.Printf("Dimension %s: %s (", name, desc)
		d := m.dimensions[name]
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
func (m *Matrix) AddDimension(name, desc string, values map[string]interface{}) {
	m.orderedDimensionNames = append(m.orderedDimensionNames, name)
	m.orderedDimensionDescs = append(m.orderedDimensionDescs, desc)
	m.dimensions[name] = values
}

// FixedDimension returns a matrixDef based on m with one of its dimensions
// fixed to a particular value. This can be used when writing tests where
// only a single value for one particular dimension is appropriate.
func (m Matrix) FixedDimension(dimensionName, valueName string) Matrix {
	return m.clone(func(dimension, value string) bool {
		return dimension != dimensionName || value == valueName
	})
}

func (m Matrix) clone(include func(dimension, value string) bool) Matrix {
	n := m
	n.dimensions = map[string]map[string]interface{}{}
	for name, values := range m.dimensions {
		nv := map[string]interface{}{}
		for vn, v := range values {
			if include(name, vn) {
				nv[vn] = v
			}
		}
		n.dimensions[name] = nv
	}
	return n
}

func (m *Matrix) scenarios() []Scenario {
	combos := [][]Scenario{}
	for _, d := range m.orderedDimensionNames {
		c := []Scenario{}
		dim := m.dimensions[d]
		valNames := []string{}
		for name := range dim {
			valNames = append(valNames, name)
		}
		sort.Strings(valNames)
		for _, name := range valNames {
			c = append(c, Scenario{
				Binding{
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

func product(slices ...[]Scenario) []Scenario {
	res := slices[0]
	for _, s := range slices[1:] {
		res = mult(res, s)
	}
	return res
}

func mult(a, b []Scenario) []Scenario {
	res := make([][]Scenario, len(a)*len(b))
	n := 0
	for _, aa := range a {
		for _, bb := range b {
			res[n] = []Scenario{aa, bb}
			n++
		}
	}
	slice := make([]Scenario, len(res))
	for i, r := range res {
		slice[i] = concat(r)
	}
	return slice
}

func concat(scenarios []Scenario) Scenario {
	res := scenarios[0]
	for _, c := range scenarios[1:] {
		res = append(res, c...)
	}
	return res
}

// String returns the sub-test path of this Scenario. E.g.
// dim1ValueName/dim2ValueName[/...].
func (c Scenario) String() string {
	var names []string
	for _, p := range c {
		names = append(names, p.Name)
	}
	return strings.Join(names, "/")
}

// Map returns a map of dimension name to specific value for this Scenario.
// This is useful where we want to look up a value by dimension name.
func (c Scenario) Map() map[string]interface{} {
	res := make(map[string]interface{}, len(c))
	for _, p := range c {
		res[p.Dimension] = p.Value
	}
	return res
}

// Value returns the value for the named dimension in this Scenario.
func (c Scenario) Value(dimension string) interface{} {
	v, ok := c.Map()[dimension]
	if !ok {
		panic(fmt.Sprintf("scenario contains no value for dimension %q", dimension))
	}
	return v
}
