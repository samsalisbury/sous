package storage

import sous "github.com/opentable/sous/lib"

// WriteState implements StateWriter on PostgresStateManager
func (m PostgresStateManager) WriteState(state *sous.State) error {
	context := context.TODO()
	tx, err := m.db.BeginTx(context, nil)
	if err != nil {
		return nil, err
	}
	defer func(tx *sql.Tx) {
		// ignoring error - since if the Tx is committed, we would expect an error on rollback
		tx.Rollback()
	}(tx)

	if err := storeManifests(context, state, tx); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return state, nil
}

func storeManifests(ctx context.Context, state *sous.State, tx *sql.Tx) {
}
