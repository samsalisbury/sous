package sous

import (
	"fmt"
	"strings"
	"testing"
)

var resolveStatusTests = []struct {
	// Phases are named "test %d; phase %d" which are 1-indexed test number, and
	// the 1-indexed phase number. See "Note 1" below.
	Phases            []interface{}
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
		Error:      "first error",
		FinalPhase: "phase 4",
	},
}

func TestResolveStatus(t *testing.T) {

	for testNum, test := range resolveStatusTests {

		// block is used to block the func passed to NewResolveStatus from
		// completing so we can run assertions that need to happen prior to
		// completion.
		block := make(chan struct{})

		// Run all the phases in the test in order.
		rs := NewResolveStatus(func(rs *ResolveStatus) {
			for phaseNum, phase := range test.Phases {
				// Note 1: 1-indexed phase naming.
				phaseName := fmt.Sprintf("test %d; phase %d", testNum+1, phaseNum+1)
				if p, ok := phase.(func()); ok {
					rs.performGuaranteedPhase(phaseName, p)
				} else if p, ok := phase.(func() error); ok {
					rs.performPhase(phaseName, p)
				} else {
					t.Fatalf("phase must be either func() or func() error")
				}
			}

			<-block // Wait for signal from the test that this func may finish.
		})

		if rs.Done() {
			t.Fatalf("Done() == true before func finished")
		}

		close(block) // Unblock the function.

		actualErr := rs.Wait()

		if !rs.Done() {
			t.Fatalf("Done() == false after Wait() call")
		}

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
	}
}
