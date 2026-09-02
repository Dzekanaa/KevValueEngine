package engine

import "fmt"

// Recover rebuilds the Memtable from the WAL on startup, replaying
// every record in order (Put for normal records, Delete for tombstones).
func (e *Engine) Recover() error {
	walRecords, err := e.wal.Recover()
	if err != nil {
		return fmt.Errorf("engine (recover): failed to read WAL: %w", err)
	}

	if len(walRecords) == 0 {
		return nil // Nothing to recover
	}

	// Replay all WAL entries to restore the Memtable state before the crash
	for _, record := range walRecords {
		if record.Tombstone {
			if err := e.memtable.Delete(string(record.Key)); err != nil {
				return fmt.Errorf("engine (recover): failed to replay delete: %w", err)
			}
		} else {
			if err := e.memtable.Put(string(record.Key), record.Value); err != nil {
				return fmt.Errorf("engine (recover): failed to replay put: %w", err)
			}
		}
	}

	return nil
}
