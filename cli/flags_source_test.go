package cli

import (
	"strings"
	"testing"

	"github.com/opentable/sous/config"
	sous "github.com/opentable/sous/lib"
)

var buildPredicateErrorTests = []struct {
	Flags config.DeployFilterFlags
	Error string
}{
	{
		Flags: config.DeployFilterFlags{
			Source: "hello",
			Repo:   "hi",
		},
		Error: "you cannot specify both -source and -repo",
	},
	{
		Flags: config.DeployFilterFlags{
			Source: "hello",
			Offset: "hi",
		},
		Error: "you cannot specify both -source and -offset",
	},
	{
		Flags: config.DeployFilterFlags{
			Source: "hello",
			All:    true,
		},
		Error: "You cannot specify both -all and filtering options",
	},
	{
		Flags: config.DeployFilterFlags{
			All:  true,
			Repo: "hello",
		},
		Error: "You cannot specify both -all and filtering options",
	},
	{
		Flags: config.DeployFilterFlags{
			All:    true,
			Offset: "hello",
		},
		Error: "You cannot specify both -all and filtering options",
	},
	{
		Flags: config.DeployFilterFlags{
			All:    true,
			Flavor: "hello",
		},
		Error: "You cannot specify both -all and filtering options",
	},
}

func TestBuildPredicate_errors(t *testing.T) {
	parseSL := func(s string) (sous.SourceLocation, error) {
		return sous.SourceLocation{Repo: "github.com/somewhere"}, nil
	}
	for _, test := range buildPredicateErrorTests {
		input := test.Flags
		expected := test.Error
		actualPredicate, actualErr := test.Flags.BuildPredicate(parseSL)
		if actualPredicate != nil {
			t.Errorf("%#v returned a non-nil predicate", input)
		}
		if actualErr == nil {
			t.Errorf("%#v returned nil error; want %q", input, expected)
			continue
		}
		actual := actualErr.Error()
		if !strings.Contains(actual, expected) {
			t.Errorf("%#v got error %q; want %q", input, actual, expected)
		}
	}
}
