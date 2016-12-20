package sous

import (
	"fmt"
	"testing"
)

var resolveStatusTests = []struct {
	Phases []interface{}
	Error  string
}{
	{
		Phases: nil,
		Error:  "",
	},
	{
		Phases: []interface{}{
			func() error {
				return nil
			},
			func() {},
		},
	},
	{
		Phases: []interface{}{
			func() error {
				return fmt.Errorf("an error")
			},
			func() {},
		},
		Error: "an error",
	},
	{
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
		Error: "first error",
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
				phaseName := fmt.Sprintf("test %d; phase %d", testNum, phaseNum)
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
		if _, errorsOpen := <-rs.Errors; errorsOpen {
			t.Errorf("Errors channel not closed")
		}
	}
}
