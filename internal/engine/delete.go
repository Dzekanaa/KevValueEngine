package engine

import "fmt"

// Delete marks key as deleted, through the WAL and then the Memtable,
// flushing first if the Memtable is full.
func (e *Engine) Delete(key string) error {
	if err := e.wal.Write([]byte(key), nil, true); err != nil {
		return fmt.Errorf("engine: delete: wal write failed: %w", err)
	}

	if err := e.memtable.Delete(key); err != nil {
		return fmt.Errorf("engine: delete: memtable delete failed: %w", err)
	}

	if e.memtable.IsFull() {
		if err := e.Flush(); err != nil {
			return fmt.Errorf("engine: delete: flush failed: %w", err)
		}
	}

	return nil
}
