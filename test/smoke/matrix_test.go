//+build smoke

package smoke

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

func TestMatrix(t *testing.T) {
	m := Matrix()
	for _, c := range m.Combinations() {
		fmt.Println(c)
	}
	m.FixtureConfigs()
}

func (m *MatrixDef) FixtureConfigs() []fixtureConfig {
	cs := m.Combinations()
	fcfgs := make([]fixtureConfig, len(cs))
	for i, c := range m.Combinations() {
		m := c.Map()
		fcfgs[i] = fixtureConfig{
			Desc:      c.String(),
			dbPrimary: m["store"].(bool),
			projects:  m["builder"].(ProjectList),
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
			v := dim[name]
			if v == nil {
				panic("nil value")
			}
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

type MatrixDef struct {
	OrderedDimensionNames []string
	Dimensions            map[string]map[string]interface{}
}

func NewMatrix() MatrixDef {
	return MatrixDef{Dimensions: map[string]map[string]interface{}{}}
}

func (m *MatrixDef) AddDimension(name string, values map[string]interface{}) {
	m.OrderedDimensionNames = append(m.OrderedDimensionNames, name)
	m.Dimensions[name] = values
}

type Particle struct {
	Dimension, Name string
	Value           interface{}
}

type Combination []Particle

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
