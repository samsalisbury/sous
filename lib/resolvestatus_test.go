package sous

import (
	"fmt"
	"strings"
	"testing"
)

var resolveStatusTests = []struct {
	// Phases are named "test %d; phase %d" which are 1-indexed test number, and
	// the 1-indexed phase number. See "Note 1" below.
	Phases      []interface{}
	Resolutions []DiffResolution

	Error, FinalPhase string
}{
	{
		// Test 1: no phases.
		FinalPhase: "finished",
	},
	{
		// Test 2: two phases.
		Phases: []interface{}{
			func() error {
				return nil
			},
			func() {},
		},
		FinalPhase: "finished",
	},
	{
		// Test 3: two phases, first fails.
		Phases: []interface{}{
			func() error {
				return fmt.Errorf("an error")
			},
			func() {},
		},
		Error:      "an error",
		FinalPhase: "phase 1",
	},
	{
		// Test 4: six phases, fourth one fails.
		Phases: []interface{}{
			func() {},
			func() {},
			func() {},
			func() error {
				return fmt.Errorf("first error")
			},
			func() error {
				return fmt.Errorf("an error")
			},
			func() {
				panic("this will not be run due to the error above")
			},
		},
		Resolutions: []DiffResolution{DiffResolution{Desc: "1"}},
		Error:       "first error",
		FinalPhase:  "phase 4",
	},
}

type rrResult struct {
	err         error
	early, late ResolveStatus
	finalPhase  string
}

func exerciseResolveRecorder(t *testing.T, f func(*ResolveRecorder)) rrResult {
	rez := rrResult{}
	block := make(chan struct{})

	// Run all the phases in the test in order.
	rs := NewResolveRecorder(NewDeployments(), func(rs *ResolveRecorder) {
		f(rs)
		close(rs.Log)
		<-block // Wait for signal from the test that this func may finish.
	})

	if rs.Done() {
		t.Fatalf("Done() == true before func finished")
	}

	rez.early = rs.CurrentStatus()

	close(block) // Unblock the function.

	rez.err = rs.Wait()

	if !rs.Done() {
		t.Fatalf("Done() == false after Wait() call")
	}

	rez.late = rs.CurrentStatus()
	rez.finalPhase = rs.Phase()

	if len(rez.early.Log) > 0 && len(rez.late.Log) > 0 {
		if &(rez.early.Log[0]) == &(rez.late.Log[0]) {
			t.Fatalf("Early and late resolution status share a log!")
		}
	}

	return rez
}

func (r rrResult) assertNoError(t *testing.T) {
	t.Helper()
	if r.err != nil {
		t.Errorf("expected no error; got error %q", r.err)
	}
}

func (r rrResult) assertError(t *testing.T, expected string) {
	t.Helper()
	if r.err == nil {
		t.Errorf("expected %q; got no error", expected)
		return
	}

	if expected != r.err.Error() {
		t.Errorf("expected %q; got %q", expected, r.err)
	}
}

func (r rrResult) assertFinalPhase(t *testing.T, expected string) {
	t.Helper()

	if expected != r.finalPhase {
		t.Errorf("expected final phase to be %q; was %q", expected, r.finalPhase)
	}
}

func (r rrResult) assertResolutionsLen(t *testing.T, expected int) {
	t.Helper()

	actual := len(r.late.Log)
	if actual != expected {
		t.Errorf("expected log of resolutions to have %d entries; has %d", expected, actual)
	}
}

func TestResolveRecorder(t *testing.T) {
	t.Run("No phases", func(t *testing.T) {
		rez := exerciseResolveRecorder(t, func(r *ResolveRecorder) {})

		rez.assertFinalPhase(t, "finished")
		rez.assertNoError(t)
		rez.assertResolutionsLen(t, 0)
	})

	t.Run("Two phases", func(t *testing.T) {
		rez := exerciseResolveRecorder(t, func(r *ResolveRecorder) {
			r.performPhase("one", func() error { return nil })
			r.performPhase("two", func() error { return nil })
		})

		rez.assertFinalPhase(t, "finished")
		rez.assertNoError(t)
		rez.assertResolutionsLen(t, 0)
	})

	t.Run("Second phase fails", func(t *testing.T) {
		rez := exerciseResolveRecorder(t, func(r *ResolveRecorder) {
			r.performPhase("one", func() error { return fmt.Errorf("an error") })
			r.performPhase("two", func() error { return nil })
		})

		rez.assertFinalPhase(t, "one")
		rez.assertError(t, "an error")
		rez.assertResolutionsLen(t, 0)
	})

	t.Run("Six phases, fourth fails", func(t *testing.T) {
		rez := exerciseResolveRecorder(t, func(r *ResolveRecorder) {
			r.performPhase("one", func() error { return nil })
			r.performPhase("two", func() error { return nil })
			r.performPhase("three", func() error {
				r.Log <- DiffResolution{Desc: "pristine"}
				return nil
			})
			r.performPhase("four -fails", func() error { return fmt.Errorf("first error") })
			r.performPhase("five -fails", func() error { return fmt.Errorf("second error") })
			r.performPhase("six -never see", func() error {
				t.Fatalf("We should never get to the sixth phase of this test.")
				return nil
			})
		})

		rez.assertError(t, "first error")
		rez.assertFinalPhase(t, "four -fails")
		rez.assertResolutionsLen(t, 1)
		if len(rez.early.Log) > 0 && len(rez.late.Log) > 0 {
			rez.early.Log[0].Desc = "mangled"
			if rez.late.Log[0].Desc != "pristine" {
				t.Errorf("Expected late statue log to be unchanged by updates to early status log: %q", rez.late.Log)
			}
		}
	})

	for testNum, test := range resolveStatusTests {

		// block is used to block the func passed to NewResolveRecorder from
		// completing so we can run assertions that need to happen prior to
		// completion.
		block := make(chan struct{})

		// Run all the phases in the test in order.
		rs := NewResolveRecorder(NewDeployments(), func(rs *ResolveRecorder) {
			performGuaranteedPhase := func(name string, f func()) {
				rs.performPhase(name, func() error {
					f()
					return nil
				})
			}
			for phaseNum, phase := range test.Phases {
				// Note 1: 1-indexed phase naming.
				phaseName := fmt.Sprintf("test %d; phase %d", testNum+1, phaseNum+1)
				if p, ok := phase.(func()); ok {
					performGuaranteedPhase(phaseName, p)
				} else if p, ok := phase.(func() error); ok {
					rs.performPhase(phaseName, p)
				} else {
					t.Fatalf("phase must be either func() or func() error")
				}
			}

			for _, rez := range test.Resolutions {
				rs.Log <- rez
			}

			// It is the responsibility of f to close the log when done.
			close(rs.Log)

			<-block // Wait for signal from the test that this func may finish.
		})

		if rs.Done() {
			t.Fatalf("Done() == true before func finished")
		}

		earlyStatus := rs.CurrentStatus()

		close(block) // Unblock the function.

		actualErr := rs.Wait()

		if !rs.Done() {
			t.Fatalf("Done() == false after Wait() call")
		}

		lateStatus := rs.CurrentStatus()

		// Assert error is correct.
		{
			expected := test.Error
			if expected == "" && actualErr != nil {
				t.Errorf("got error %q; want nil", actualErr)
			} else if expected != "" && actualErr == nil {
				t.Errorf("got nil; want error %q", expected)
			} else if actualErr != nil {
				if actual := actualErr.Error(); actual != expected {
					t.Errorf("got error %q; want %q", actual, expected)
				}
			}
		}

		// Assert final phase has correct suffix, see "Note 1" above.
		{
			expected := test.FinalPhase
			actual := rs.Phase()
			if !strings.HasSuffix(actual, expected) {
				t.Errorf("final phase == %q; want suffix %q", actual, expected)
			}
		}

		{
			expected := test.Resolutions
			actual := lateStatus.Log
			if len(actual) != len(expected) {
				t.Errorf("final log of resolutions has wrong number of entries %d vs %d", len(expected), len(actual))
			}
		}

		{
			if len(earlyStatus.Log) > 0 {
				earlyStatus.Log[0].Desc = "changed"
				// ugh, this is kind of suspect...
				if lateStatus.Log[0].Desc != test.Resolutions[0].Desc {
					t.Errorf("early  and late statuses share a log!")
				}
			}
		}
	}
}
