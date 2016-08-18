package cmap

import (
	"sync"
	"testing"
)

type CMapTest struct {
	CMap     CMap
	Do       func(c *CMap)
	Expected map[CMKey]Value
}

var cmapTests = []CMapTest{
	{
		CMap:     NewCMap(),
		Expected: map[CMKey]Value{},
	},
	{
		CMap:     MakeCMap(999),
		Expected: NewCMap().Snapshot(),
	},
	{
		CMap:     NewCMap(),
		Expected: NewCMap().Snapshot(),
	},
	{
		CMap:     NewCMap("one"),
		Expected: map[CMKey]Value{"one": "one"},
	},
	{
		CMap:     NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Expected: map[CMKey]Value{"one": "one"},
	},
	{
		CMap:     NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Expected: map[CMKey]Value{"one": "one"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Do: func(c *CMap) {
			*c = c.Filter(func(v Value) bool { return v.ID() == "one" })
		},
		Expected: map[CMKey]Value{"one": "one"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Do: func(c *CMap) {
			*c = c.Filter(nil)
		},
		Expected: map[CMKey]Value{"one": "one"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Do: func(c *CMap) {
			*c = c.Filter(func(v Value) bool { return v.ID() == "two" })
		},
		Expected: map[CMKey]Value{},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Do: func(c *CMap) {
			ok := c.Add("two")
			if !ok {
				panic("failed to add two")
			}
		},
		Expected: map[CMKey]Value{"one": "one", "two": "two"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Do: func(c *CMap) {
			ok := c.Add("one")
			if ok {
				panic("added one twice")
			}
		},
		Expected: map[CMKey]Value{"one": "one"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Do: func(c *CMap) {
			c.MustAdd("two")
		},
		Expected: map[CMKey]Value{"one": "one", "two": "two"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Do: func(c *CMap) {
			c.AddAll(NewCMap("two", "three"))
		},
		Expected: map[CMKey]Value{"one": "one", "two": "two", "three": "three"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Do: func(c *CMap) {
			c.AddAll(NewCMap("two", "three"))
			c.Remove("two")
		},
		Expected: map[CMKey]Value{"one": "one", "three": "three"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Do: func(c *CMap) {
			*c = NewCMapFromMap(c.FilteredSnapshot(func(v Value) bool {
				return v == "two"
			}))
		},
		Expected: map[CMKey]Value{},
	},
	{
		CMap: NewCMap(),
		Do: func(c *CMap) {
			c.SetAll(map[CMKey]Value{"set": "set", "all": "all"})
		},
		Expected: map[CMKey]Value{"set": "set", "all": "all"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"set": "set", "all": "all"}),
		Do: func(c *CMap) {
			*c = NewCMapFromMap(c.GetAll())
		},
		Expected: map[CMKey]Value{"set": "set", "all": "all"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"set": "set", "all": "all"}),
		Do: func(c *CMap) {
			*c = c.Clone()
		},
		Expected: map[CMKey]Value{"set": "set", "all": "all"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one", "two": "two"}),
		Do: func(c *CMap) {
			v, ok := c.Get("two")
			if !ok {
				panic("missing key two")
			}
			*c = NewCMap(v)
		},
		Expected: map[CMKey]Value{"two": "two"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{}),
		Do: func(c *CMap) {
			c.Set("two", "two")
		},
		Expected: map[CMKey]Value{"two": "two"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one", "two": "two"}),
		Do: func(c *CMap) {
			v, ok := c.Single(func(v Value) bool {
				return v == "two"
			})
			if !ok {
				panic("no single value two")
			}
			*c = NewCMap(v)
		},
		Expected: map[CMKey]Value{"two": "two"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one", "two": "two"}),
		Do: func(c *CMap) {
			v, ok := c.Single(func(v Value) bool {
				return v == "nonexistent"
			})
			if ok {
				panic("found nonexistent value")
			}
			*c = NewCMap(v)
		},
		Expected: map[CMKey]Value{"": ""},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one", "two": "two"}),
		Do: func(c *CMap) {
			v, ok := c.Any(func(v Value) bool {
				return v == "two"
			})
			if !ok {
				panic("no value two")
			}
			*c = NewCMap(v)
		},
		Expected: map[CMKey]Value{"two": "two"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one", "two": "two"}),
		Do: func(c *CMap) {
			v, ok := c.Any(func(v Value) bool {
				return v == "nonexistent"
			})
			if ok {
				panic("found value; should not have")
			}
			*c = NewCMap(v)
		},
		Expected: map[CMKey]Value{"": ""},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Do: func(c *CMap) {
			other := NewCMapFromMap(map[CMKey]Value{"two": "two"})
			*c = c.Merge(other)
		},
		Expected: map[CMKey]Value{"one": "one", "two": "two"},
	},
	{
		CMap: NewCMapFromMap(map[CMKey]Value{"one": "one"}),
		Do: func(c *CMap) {
			vals := []Value{}
			for _, k := range c.Keys() {
				vals = append(vals, Value(string(k)))
			}
			other := NewCMap(vals...)
			*c = c.Merge(other)
		},
		Expected: map[CMKey]Value{"one": "one"},
	},
}

func TestCMap(t *testing.T) {

	wg := sync.WaitGroup{}
	wg.Add(len(cmapTests))
	for _, test := range cmapTests {
		test := test
		go func() {
			defer wg.Done()
			if test.Do != nil {
				test.Do(&test.CMap)
			}
			actual := test.CMap.Snapshot()
			expected := test.Expected
			if test.CMap.Len() != len(actual) {
				t.Errorf("Len ≠ len")
			}
			if len(actual) != len(expected) {
				t.Errorf("got len %d; want %d", len(actual), len(expected))
			}
			for actualKey := range actual {
				if _, ok := expected[actualKey]; !ok {
					t.Errorf("extra key %q", actualKey)
				}
			}
			for expectedKey := range expected {
				if _, ok := actual[expectedKey]; !ok {
					t.Errorf("missing key %q", expectedKey)
				}
			}
		}()
	}
	wg.Wait()

}
