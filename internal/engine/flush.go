package engine

import "fmt"

// Flush persists the current Memtable contents to a new SSTable, then
// clears the Memtable, then prunes WAL segments whose data is now
// durably stored elsewhere.
//
// Order matters and must not change: the SSTable write must succeed
// before the Memtable is cleared or the WAL is pruned. If the process
// crashes between steps, the WAL still holds everything needed to
// recover — but only as long as it hasn't been pruned yet.
func (e *Engine) Flush() error {
	entries := e.memtable.GetSorted()
	if len(entries) == 0 {
		return nil
	}

	if _, err := e.writer.Write(entries); err != nil {
		return fmt.Errorf("engine: flush: sstable write failed: %w", err)
	}

	e.memtable.Clear()

	if err := e.wal.Cleanup(); err != nil {
		return fmt.Errorf("engine: flush: wal cleanup failed: %w", err)
	}

	return nil
}
