// +build shelltesttest

package shelltest

import "testing"

func TestShellTest(t *testing.T) {
	sh := New(t)

	simple := sh.Block("simple",
		`echo testing`,
		func(res Result, t *testing.T) {
			if !res.Matches(`testing`) {
				t.Error("no testing!")
			}
		})

	fails := simple.Block("fails on purpose",
		`false`,
		func(res Result, t *testing.T) {
			if res.Exit != 0 {
				t.Error("I expected that")
			}
		})
	fails.Block("shouldn't run",
		`echo AMAZING`,
		func(res Result, t *testing.T) {
			panic("never happen")
		})

	simple.Block("failnow but keep going",
		`echo whatever`,
		func(res Result, t *testing.T) {
			t.Fatal("can't be bothered")
			panic("already failed")
		})

	simple.Block("safe home",
		`echo $HOME`,
		func(res Result, t *testing.T) {})
}
