//+build smoke

package smoke

import (
	"fmt"
	"sort"
	"strings"
)

type MatrixDef struct {
	OrderedDimensionNames []string
	OrderedDimensionDescs []string
	Dimensions            map[string]map[string]interface{}
}

type Combination []Particle

type Particle struct {
	Dimension, Name string
	Value           interface{}
}

func NewMatrix() MatrixDef {
	return MatrixDef{Dimensions: map[string]map[string]interface{}{}}
}

func (m MatrixDef) PrintDimensions() {
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

func (m *MatrixDef) AddDimension(name, desc string, values map[string]interface{}) {
	m.OrderedDimensionNames = append(m.OrderedDimensionNames, name)
	m.OrderedDimensionDescs = append(m.OrderedDimensionDescs, desc)
	m.Dimensions[name] = values
}

// TODO SS: Remove this from MatrixDef and write a helper func to do the same.
func (m *MatrixDef) FixtureConfigs() []fixtureConfig {
	cs := m.Combinations()
	fcfgs := make([]fixtureConfig, len(cs))
	for i, c := range m.Combinations() {
		m := c.Map()
		fcfgs[i] = fixtureConfig{
			Desc:      c.String(),
			dbPrimary: m["store"].(bool),
			projects:  m["project"].(ProjectList),
		}
	}
	return fcfgs
}

func (m *MatrixDef) Combinations() []Combination {
	combos := [][]Combination{}
	for _, d := range m.OrderedDimensionNames {
		c := []Combination{}
		dim := m.Dimensions[d]
		valNames := []string{}
		for name := range dim {
			valNames = append(valNames, name)
		}
		sort.Strings(valNames)
		for _, name := range valNames {
			c = append(c, Combination{
				Particle{
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

func product(slices ...[]Combination) []Combination {
	res := slices[0]
	for _, s := range slices[1:] {
		res = mult(res, s)
	}
	return res
}

func mult(a, b []Combination) []Combination {
	res := make([][]Combination, len(a)*len(b))
	n := 0
	for _, aa := range a {
		for _, bb := range b {
			res[n] = []Combination{aa, bb}
			n++
		}
	}
	slice := make([]Combination, len(res))
	for i, r := range res {
		slice[i] = concat(r)
	}
	return slice
}

func concat(combos []Combination) Combination {
	res := combos[0]
	for _, c := range combos[1:] {
		res = append(res, c...)
	}
	return res
}

func (c Combination) String() string {
	var names []string
	for _, p := range c {
		names = append(names, p.Name)
	}
	return strings.Join(names, "/")
}

func (c Combination) Map() map[string]interface{} {
	res := make(map[string]interface{}, len(c))
	for _, p := range c {
		res[p.Dimension] = p.Value
	}
	return res
}
