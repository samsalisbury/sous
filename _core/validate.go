package core

import "fmt"

func (s *State) Validate() error {
	// Check that none of the manifests overwrite the protected
	// env vars.
	for _, manifest := range s.Manifests {
		for _, deployment := range manifest.Deployments {
			for _, envVar := range *s.EnvironmentDefs["Universal"] {
				if _, exists := deployment.Environment[envVar.Name]; exists {
					return fmt.Errorf(
						"%s overrides protected environment variable %s",
						manifest.App.SourceRepo, envVar)
				}
			}
		}
	}
	return nil
}

func (c *Contract) ValidateTest() error {
	e := func(f string, a ...interface{}) error {
		return fmt.Errorf("contract test %q %s", c.Name,
			fmt.Sprintf(f, a...))
	}
	numTests := len(c.SelfTest.CheckTests)
	numChecks := len(c.Checks)
	if numTests != numChecks {
		return e("has %d check tests; want %d", numTests, numChecks)
	}
	// Check that each check has a test in the right order, with both
	// pass and fail specified.
	for i, check := range c.Checks {
		te := func(f string, a ...interface{}) error {
			return e("check test at position %d %s", i, fmt.Sprintf(f, a...))
		}
		test := c.SelfTest.CheckTests[i]
		if test.CheckName != check.Name {
			return te("got test for %q; want %q", test.CheckName, check.Name)
		}
		if test.TestImages.Pass == "" {
			return te("missing Pass image")
		}
		if test.TestImages.Fail == "" {
			return te("missing Fail image")
		}
	}
	return nil
}
