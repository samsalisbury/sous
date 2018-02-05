package sous

import (
	"fmt"
	"testing"
)

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
}
