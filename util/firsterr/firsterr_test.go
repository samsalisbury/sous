package firsterr

import "testing"

type Error string

func (err Error) Error() string { return string(err) }

func setErr(name string) func(*error)    { return func(err *error) { *err = Error(name) } }
func setNil(err *error)                  { *err = nil }
func returnErr(name string) func() error { return func() error { return Error(name) } }
func returnNil() error                   { return nil }

func (s S) testSet(fs ...func(*error)) setTest       { return setTest{s.Set, fs} }
func (p P) testSet(fs ...func(*error)) setTest       { return setTest{p.Set, fs} }
func (s S) testReturn(fs ...func() error) returnTest { return returnTest{s.Returned, fs} }
func (p P) testReturn(fs ...func() error) returnTest { return returnTest{p.Returned, fs} }

type setTest struct {
	set func(...func(*error)) error
	fs  []func(*error)
}

type returnTest struct {
	returned func(...func() error) error
	fs       []func() error
}

func (st setTest) shouldReturn(expected error) error {
	return assert(st.set(st.fs...), expected)
}

func (rt returnTest) shouldReturn(expected error) error {
	return assert(rt.returned(rt.fs...), expected)
}

func assert(actual, expected error) error {
	if actual == nil && expected == nil {
		return nil
	}
	if actual != nil && expected != nil && actual.Error() == expected.Error() {
		return nil
	}
	if expected == nil {
		return Error("error was \"" + actual.Error() + "\"; want nil")
	}
	if actual == nil {
		return Error("error was nil; want \"" + expected.Error() + "\"")
	}
	return Error("error was \"" + actual.Error() +
		"\"; want \"" + expected.Error() + "\"")
}

func TestSequential_Set_Err(t *testing.T) {
	// Check we get back expected errors...
	var testErrs = []error{
		s.testSet(setNil).shouldReturn(nil),
		s.testSet(setErr("one")).shouldReturn(Error("one")),
		s.testSet(setErr("one"), setErr("two")).shouldReturn(Error("one")),
		s.testSet(setNil, setErr("one"), setErr("two")).shouldReturn(Error("one")),
		s.testSet(setNil, setErr("two"), setErr("one")).shouldReturn(Error("two")),
		s.testSet(setErr("two"), setErr("one"), setNil).shouldReturn(Error("two")),
	}
	for _, testErr := range testErrs {
		if testErr != nil {
			t.Error(testErr)
		}
	}

	// Check functions are actually run...
	callCount := 0
	spy := func(*error) { callCount++ }
	s.Set(spy, spy, spy, spy, spy)
	if callCount != 5 {
		t.Fatalf("spy called %d times; want 5", callCount)
	}

	// Check functions after first failure are not run...
	callCount = 0
	s.Set(spy, spy, spy, setErr("one"), spy, spy)
	if callCount != 3 {
		t.Fatalf("spy called %d times; want 3", callCount)
	}
}

func TestSequential_Return_Err(t *testing.T) {
	// Check we get back expected errors...
	var testErrs = []error{
		s.testReturn(returnNil).shouldReturn(nil),
		s.testReturn(returnErr("one")).shouldReturn(Error("one")),
		s.testReturn(returnErr("one"), returnErr("two")).shouldReturn(Error("one")),
		s.testReturn(returnNil, returnErr("one"), returnErr("two")).shouldReturn(Error("one")),
		s.testReturn(returnNil, returnErr("two"), returnErr("one")).shouldReturn(Error("two")),
		s.testReturn(returnErr("two"), returnErr("one"), returnNil).shouldReturn(Error("two")),
	}
	for _, testErr := range testErrs {
		if testErr != nil {
			t.Error(testErr)
		}
	}

	// Check functions are actually run...
	callCount := 0
	spy := func() error { callCount++; return nil }
	s.Returned(spy, spy, spy, spy, spy)
	if callCount != 5 {
		t.Fatalf("spy called %d times; want 5", callCount)
	}

	// Check functions after first failure are not run...
	callCount = 0
	s.Returned(spy, spy, spy, returnErr("one"), spy, spy)
	if callCount != 3 {
		t.Fatalf("spy called %d times; want 3", callCount)
	}
}

func TestParallel_Set_Err(t *testing.T) {
	// Check we get back expected errors...
	var testErrs = []error{
		p.testSet(setNil).shouldReturn(nil),
		p.testSet(setErr("one")).shouldReturn(Error("one")),
		p.testSet(setErr("one"), setErr("one")).shouldReturn(Error("one")),
		p.testSet(setNil, setErr("one"), setErr("one")).shouldReturn(Error("one")),
		p.testSet(setErr("two"), setErr("two"), setNil).shouldReturn(Error("two")),
	}
	for _, testErr := range testErrs {
		if testErr != nil {
			t.Error(testErr)
		}
	}

	// Check functions are actually run...
	callCount := 0
	spy := func(*error) { callCount++ }
	p.Set(spy, spy, spy, spy, spy)
	if callCount != 5 {
		t.Fatalf("spy called %d times; want 5", callCount)
	}
}

func TestParallel_Return_Err(t *testing.T) {
	// Check we get back expected errors...
	var testErrs = []error{
		p.testReturn(returnNil).shouldReturn(nil),
		p.testReturn(returnErr("one")).shouldReturn(Error("one")),
		p.testReturn(returnErr("one"), returnErr("one")).shouldReturn(Error("one")),
		p.testReturn(returnNil, returnErr("one"), returnErr("one")).shouldReturn(Error("one")),
		p.testReturn(returnErr("two"), returnErr("two"), returnNil).shouldReturn(Error("two")),
	}
	for _, testErr := range testErrs {
		if testErr != nil {
			t.Error(testErr)
		}
	}

	// Check all functions are run when there is no error.
	callCount := 0
	spy := func() error { callCount++; return nil }
	s.Returned(spy, spy, spy, spy, spy)
	if callCount != 5 {
		t.Fatalf("spy called %d times; want 5", callCount)
	}
}
