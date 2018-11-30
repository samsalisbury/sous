package graph

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/samsalisbury/psyringe"
)

func TestNewPrimaryStateManager(t *testing.T) {

	gitSM, distSM := twoDistinctStateManagers(t)

	result := func(t *testing.T, c config.Config, gitErr, distErr error) (sous.StateManager, error) {
		t.Helper()
		git := gitStateManager{StateManager: gitSM, Error: gitErr}
		dist := DistStateManager{StateManager: distSM, Error: distErr}
		lsc := LocalSousConfig{Config: &c}
		return newPrimaryStateManager(lsc, git, dist)
	}
	assertPrimary := func(t *testing.T, c config.Config, gitErr, distErr error, want sous.StateManager) {
		t.Helper()
		got, err := result(t, c, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !sameStateManagers(t, got, want) {
			t.Errorf("got wrong state manager")
		}
	}
	assertErr := func(t *testing.T, c config.Config, gitErr, distErr error, want string) {
		t.Helper()
		_, err := result(t, c, gitErr, distErr)
		if err == nil {
			t.Fatalf("got nil; want error %q", want)
		}
		got := err.Error()
		if got != want {
			t.Errorf("got error %q; want %q", got, want)
		}
	}

	t.Run("emptycfg->git", func(t *testing.T) {
		c := config.Config{}
		assertPrimary(t, c, nil, nil, gitSM)
	})
	t.Run("dbprimary->dist", func(t *testing.T) {
		c := config.Config{DatabasePrimary: true}
		assertPrimary(t, c, nil, nil, distSM)
	})
	t.Run("emptycfg,dberr->git", func(t *testing.T) {
		c := config.Config{}
		assertPrimary(t, c, nil, fmt.Errorf("dberr"), gitSM)
	})
	t.Run("dbprimary,giterr->dist", func(t *testing.T) {
		c := config.Config{DatabasePrimary: true}
		assertPrimary(t, c, fmt.Errorf("giterr"), nil, distSM)
	})
	t.Run("emptycfg,giterr->err", func(t *testing.T) {
		c := config.Config{}
		assertErr(t, c, fmt.Errorf("giterr"), nil, "giterr")
	})
	t.Run("dbprimary,dberr->err", func(t *testing.T) {
		c := config.Config{DatabasePrimary: true}
		assertErr(t, c, nil, fmt.Errorf("dberr"), "dberr")
	})
}

func TestNewSecondaryStateManager(t *testing.T) {

	gitSM, distSM := twoDistinctStateManagers(t)

	result := func(t *testing.T, c config.Config, gitErr, distErr error) sous.StateManager {
		t.Helper()
		git := gitStateManager{StateManager: gitSM, Error: gitErr}
		dist := DistStateManager{StateManager: distSM, Error: distErr}
		lsc := LocalSousConfig{Config: &c}
		ls := LogSink{LogSink: logging.SilentLogSet()}
		return newSecondaryStateManager(ls, lsc, git, dist)
	}
	assertSecondary := func(t *testing.T, c config.Config, gitErr, distErr error, want sous.StateManager) {
		t.Helper()
		got := result(t, c, gitErr, distErr)
		if !sameStateManagers(t, got, want) {
			t.Errorf("got wrong state manager")
		}
	}
	assertLogOnly := func(t *testing.T, c config.Config, gitErr, distErr error) {
		t.Helper()
		got := result(t, c, gitErr, distErr)
		gotT := reflect.TypeOf(got)
		wantT := reflect.TypeOf(&storage.LogOnlyStateManager{})
		if gotT != wantT {
			t.Errorf("got type %s; want %s", gotT, wantT)
		}
	}

	t.Run("emptycfg->dist", func(t *testing.T) {
		c := config.Config{}
		assertSecondary(t, c, nil, nil, distSM)
	})
	t.Run("dbprimary->git", func(t *testing.T) {
		c := config.Config{DatabasePrimary: true}
		assertSecondary(t, c, nil, nil, gitSM)
	})
	t.Run("emptycfg,dberr->logonly", func(t *testing.T) {
		c := config.Config{}
		assertLogOnly(t, c, nil, fmt.Errorf("dberr"))
	})
	t.Run("dbprimary,giterr->logonly", func(t *testing.T) {
		c := config.Config{DatabasePrimary: true}
		assertLogOnly(t, c, fmt.Errorf("giterr"), nil)
	})
}

func TestNewDuplexStateManager_ok(t *testing.T) {

	primary, secondary := twoDistinctStateManagers(t)

	got := newDuplexStateManager(primary, secondary)

	if !sameStateManagers(t, got.primary, primary) {
		t.Errorf("primary wrong")
	}
	if !sameStateManagers(t, got.secondary, secondary) {
		t.Errorf("secondary wrong")
	}
}

// TestNewDuplexStateManager_ok tests the interaction of constructors leading to
// and eventual server state manager.
func TestNewDuplexStateManager_integration(t *testing.T) {

	dbType := reflect.TypeOf(&storage.PostgresStateManager{})
	gitType := reflect.TypeOf(&storage.GitStateManager{})
	httpType := reflect.TypeOf(&sous.HTTPStateManager{})
	dispatchType := reflect.TypeOf(&sous.DispatchStateManager{})
	logOnlyType := reflect.TypeOf(&storage.LogOnlyStateManager{})
	t.Log("types:", dbType, gitType, httpType, logOnlyType, dispatchType)

	type testCase struct {
		cfg           config.Config
		gitErr, dbErr error
	}
	type result struct {
		primary, secondary reflect.Type
	}
	assertPrimarySecondary := func(tc testCase, want result) func(*testing.T) {
		return func(t *testing.T) {

			g := psyringe.New(
				// Everything depends on logging.
				LogSink{logging.SilentLogSet()},

				// Constructors under test.
				newDuplexStateManager,
				newPrimaryStateManager,
				newSecondaryStateManager,

				// Things that change per test case.

				LocalSousConfig{Config: &tc.cfg},
				gitStateManager{
					StateManager: &storage.GitStateManager{},
					Error:        tc.gitErr,
				},
				DistStateManager{
					StateManager: &sous.DispatchStateManager{},
					Error:        tc.dbErr,
				},
			)

			scoop := struct {
				Got duplexStateManager
			}{}

			if err := g.Inject(&scoop); err != nil {
				t.Fatal(err)
			}
			got := result{
				primary:   reflect.TypeOf(scoop.Got.primary),
				secondary: reflect.TypeOf(scoop.Got.secondary),
			}

			if got.primary != want.primary {
				t.Errorf("got primary %s; want %s", got.primary, want.primary)
			}

			if got.secondary != want.secondary {
				t.Errorf("got secondary %s; want %s", got.secondary, want.secondary)
			}
		}
	}

	t.Run("emptycfg", assertPrimarySecondary(
		testCase{
			cfg: config.Config{},
		},
		result{primary: gitType, secondary: dispatchType}),
	)
	t.Run("emptycfg,dberr", assertPrimarySecondary(
		testCase{
			cfg:   config.Config{},
			dbErr: fmt.Errorf("dberr"),
		},
		result{primary: gitType, secondary: logOnlyType}),
	)
	t.Run("dbprimary", assertPrimarySecondary(
		testCase{
			cfg: config.Config{DatabasePrimary: true},
		},
		result{primary: dispatchType, secondary: gitType}),
	)
	t.Run("dbprimary,giterr", assertPrimarySecondary(
		testCase{
			cfg:    config.Config{DatabasePrimary: true},
			gitErr: fmt.Errorf("giterr"),
		},
		result{primary: dispatchType, secondary: logOnlyType}),
	)

}

// equalStates returns true when 2 states have different number of clusters.
// This is to be used only for comparing states made with differentStates and
// twoDistinctStates.
func equalStates(a, b *sous.State) bool {
	return len(a.Defs.Clusters) == len(b.Defs.Clusters)
}
func differentStates(count int) []*sous.State {
	var out []*sous.State
	for i := 0; i < count; i++ {
		out = append(out, sous.StateFixture(sous.StateFixtureOpts{
			ClusterCount: i,
		}))
	}
	return out
}

// twoDistinctStates returns 2 states that are distinct in that calling
// equalStates and passing them both in will return false.
func twoDistinctStates(t *testing.T) (state1, state2 *sous.State) {
	t.Helper()
	states := differentStates(2)
	state1, state2 = states[0], states[1]
	if equalStates(state1, state2) {
		t.Fatal("Test setup failed to produce distinct states.")
	}
	return
}

func twoDistinctStateManagers(t *testing.T) (sm1, sm2 sous.StateManager) {
	t.Helper()
	a, b := twoDistinctStates(t)
	sm1 = &sous.DummyStateManager{State: a}
	sm2 = &sous.DummyStateManager{State: b}
	if sameStateManagers(t, sm1, sm2) {
		t.Fatalf("Test setup faile top produce distinct state managers.")
	}
	return
}

func sameStateManagers(t *testing.T, a, b sous.StateManager) bool {
	t.Helper()
	as, err := a.ReadState()
	if err != nil {
		t.Fatal(err)
	}
	bs, err := b.ReadState()
	if err != nil {
		t.Fatal(err)
	}
	return equalStates(as, bs)
}
