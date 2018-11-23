package graph

import (
	"database/sql"
	"fmt"

	"github.com/opentable/sous/ext/storage"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/pkg/errors"
)

type (
	// ClusterManager simply wraps the sous.ClusterManager interface
	ClusterManager struct{ sous.ClusterManager }
	// ClientStateManager wraps the sous.StateManager interface and is used by non-server sous commands
	ClientStateManager struct{ sous.StateManager }
	// ServerStateManager wraps the sous.StateManager interface and is used by `sous server`
	ServerStateManager struct{ sous.StateManager }
	// ServerClusterManager wraps the sous.ClusterManager interface and is used by `sous server`
	ServerClusterManager struct{ sous.ClusterManager }
	// DistStateManager wraps sous.StateManager interfaces and is used by `sous server`
	stateManagerAndErr struct {
		sous.StateManager
		Error error
	}

	// DistStateManager contains a distributed state manager and the error
	// returned from its construction.
	DistStateManager stateManagerAndErr
	gitStateManager  stateManagerAndErr

	primaryStateManager   sous.StateManager
	secondaryStateManager sous.StateManager
)

// newClientStateManager returns a wrapped sous.HTTPStateManager if cl is not nil.
// Otherwise it returns a wrapped sous.GitStateManager, for local git based GDM.
// If it returns a sous.GitStateManager, it emits a warning log.
func newClientStateManager(cl HTTPClient, c LocalSousConfig, mdb MaybeDatabase, tid sous.TraceID, rf *sous.ResolveFilter, log LogSink) (*ClientStateManager, error) {
	if c.Server == "" {
		return nil, errors.New("no server configured for state management")
	}
	hsm := sous.NewHTTPStateManager(cl, tid, log.Child("http-state-manager"))
	return &ClientStateManager{StateManager: hsm}, nil
}

func newServerStateManager(log LogSink, primary primaryStateManager, secondary secondaryStateManager) *ServerStateManager {
	return &ServerStateManager{
		StateManager: storage.NewDuplexStateManager(
			primary, secondary, log.Child("duplex-state"),
		),
	}
}

func newHTTPStateManager(cl HTTPClient, tid sous.TraceID, log LogSink) *sous.HTTPStateManager {
	return sous.NewHTTPStateManager(cl, tid, log.Child("http-state-manager"))
}

// newPrimaryStateManager returns the configured primary, and any error
// encountered in its construction.
func newPrimaryStateManager(c LocalSousConfig, gm gitStateManager, dm DistStateManager) (primaryStateManager, error) {
	if c.DatabasePrimary {
		return dm.StateManager, dm.Error
	}
	return gm.StateManager, gm.Error
}

// newSecondaryStateManager returns the configured secondary state manager if
// there were no errors constructing it. Otherwise it emits a log message with
// the error and falls back to the log-only state manager.
func newSecondaryStateManager(log LogSink, c LocalSousConfig, gm gitStateManager, dm DistStateManager) secondaryStateManager {
	var sm sous.StateManager
	var err error
	var name string
	if c.DatabasePrimary {
		name, sm, err = "db", dm.StateManager, dm.Error
	} else {
		name, sm, err = "git", gm.StateManager, gm.Error
	}
	if err == nil {
		return sm
	}
	logging.WarnMsg(log, "secondary state manager %q unavailable: %s", name, err)
	logging.WarnMsg(log, "secondary state manager: falling back to log-only")
	return storage.NewLogOnlyStateManager(log.Child("log-only-statemanager"))
}

func newServerClusterManager(c LocalSousConfig, log LogSink, primary primaryStateManager) (*ServerClusterManager, error) {
	return &ServerClusterManager{ClusterManager: sous.MakeClusterManager(primary, log)}, nil
}

func newDistributedStateManager(c LocalSousConfig, mdb MaybeDatabase, tid sous.TraceID, rf *sous.ResolveFilter, log LogSink) DistStateManager {
	var dist sous.StateManager
	err := mdb.Err
	if err == nil {
		dist, err = newDistributedStorage(mdb.Db, c, tid, rf, log)
	}

	return DistStateManager{
		StateManager: dist,
		Error:        err,
	}
}

func newGitStateManager(dm *storage.DiskStateManager, log LogSink) gitStateManager {
	return gitStateManager{StateManager: storage.NewGitStateManager(dm, log.Child("git-state-manager"))}
}

func newDiskStateManager(c LocalSousConfig, log LogSink) *storage.DiskStateManager {
	return storage.NewDiskStateManager(c.StateLocation, log.Child("disk-state-manager"))
}

func newDistributedStorage(db *sql.DB, c LocalSousConfig, tid sous.TraceID, rf *sous.ResolveFilter, log LogSink) (sous.StateManager, error) {
	localName, err := rf.Cluster.Value()
	if err != nil {
		return nil, fmt.Errorf("Setting up distributed storage: cluster: %s", err) // errors.Wrapf && cli don't play nice
	}

	local := storage.NewPostgresStateManager(db, log.Child("database"))
	list := ClientBundle{}
	clusterNames := []string{}
	for n, u := range c.SiblingURLs {
		// XXX not immediately clear how to conserve the request id through the distributed storage.
		cl, err := restful.NewClient(u, log.Child(n+".http-client"))
		if err != nil {
			return nil, err
		}
		list[n] = cl
		clusterNames = append(clusterNames, n)
	}
	// XXX the first arg is used to get e.g. defs. Should be at least an in memory client for these purposes.
	hsm := sous.NewHTTPStateManager(list[localName], tid, log.Child("http-state-manager"))
	return sous.NewDispatchStateManager(localName, clusterNames, local, hsm, log.Child("state-manager")), nil
}
