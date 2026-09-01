package engine

import "fmt"

// Recover rebuilds the Memtable from the WAL on startup, replaying
// every record in order (Put for normal records, Delete for tombstones).
//
// Marija i Niksa
// TODO: implement this once Flush is working
// end to end.
func (e *Engine) Recover() error {
	walRecords, err := e.wal.ReadAll()
	if err != nil {
		return fmt.Errorf("engine (recover): failed to read WAL: %w", err)
	}

	if len(walRecords) == 0 {
		return nil // Nothing to recover
	}

	// Converts wal.Record into engine.WALRecord
	records := make([]WALRecord, len(walRecords))
	for i, r := range walRecords {
		records[i] = WALRecord{
			Key:       r.Key,
			Value:     r.Value,
			Tombstone: r.Tombstone,
		}
	}

	// TODO: add comment
	for _, record := range records {
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

	// Marija
	// TODO: Mark which WAL segments need to be deleted

	return nil
}
