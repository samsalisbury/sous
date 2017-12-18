package configloader

import (
	"os"
	"testing"
)

type TestConfig struct {
	SomeVar string `env:"TEST_SOME_VAR"`
	TestedNested
}

type TestedNested struct {
	NestedVar string `env:"TEST_NESTED_VAR"`
}

type TestedMap struct {
	SiblingURLs map[string]string `env:"TEST_MAP"`
}

func (tc *TestConfig) FillDefaults() error {
	if tc.SomeVar == "" {
		tc.SomeVar = "default value"
	}
	return nil
}

func TestLoad(t *testing.T) {
	cl := New()
	c := TestConfig{}
	if err := cl.Load(&c, "testdata/test_config.yaml"); err != nil {
		t.Fatal(err)
	}
	expected := "some value"
	if c.SomeVar != expected {
		t.Errorf("got SomeVar=%q; want %q", c.SomeVar, expected)
	}
}

func TestLoad_Defaults(t *testing.T) {
	cl := New()
	c := TestConfig{}
	if err := cl.Load(&c, "testdata/test_empty_config.yaml"); err != nil {
		t.Fatal(err)
	}
	expected := "default value"
	if c.SomeVar != expected {
		t.Errorf("got SomeVar=%q; want %q", c.SomeVar, expected)
	}
}

func TestLoad_Env(t *testing.T) {
	cl := New()
	c := TestConfig{}

	expected := "other value"
	os.Setenv("TEST_SOME_VAR", expected)
	os.Setenv("TEST_NESTED_VAR", expected)

	if e := os.Getenv("TEST_SOME_VAR"); e != expected {
		t.Fatalf("setenv failed")
	}

	if err := cl.Load(&c, "testdata/test_config.yaml"); err != nil {
		t.Fatal(err)
	}

	if c.SomeVar != expected {
		t.Errorf("got SomeVar=%q; want %q", c.SomeVar, expected)
	}

	if c.TestedNested.NestedVar != expected {
		t.Errorf("got NestedVar=%q; want %q", c.TestedNested.NestedVar, expected)
	}
}

func TestLoad_EmptyEnv(t *testing.T) {
	cl := New()
	c := TestConfig{}

	expected := ""
	os.Setenv("TEST_SOME_VAR", expected)

	if e := os.Getenv("TEST_SOME_VAR"); e != expected {
		t.Fatalf("setenv failed")
	}

	if err := cl.Load(&c, "testdata/test_config.yaml"); err != nil {
		t.Fatal(err)
	}

	if c.SomeVar != expected {
		t.Errorf("got SomeVar=%q; want %q", c.SomeVar, expected)
	}
}

func TestLoad_Map(t *testing.T) {
	cl := New()
	c := TestedMap{}

	s := `{"env1": "foo", "env2": "bar"}`

	os.Setenv("TEST_MAP", s)

	if err := cl.Load(&c, "testdata/test_map_config.yaml"); err != nil {
		t.Fatal(err)
	}

	if _, ok := c.SiblingURLs["env3"]; ok {
		t.Errorf("Failed to override map from environment variable")
	}

	if val := c.SiblingURLs["env1"]; val != "foo" {
		t.Errorf("Value not expected")
	}
}
