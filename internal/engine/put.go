package engine

import "fmt"

// Put writes key through the WAL, then applies it to the Memtable,
// flushing first if the Memtable is full as a result.
//
// TODO: implement.
func (e *Engine) Put(key string, value []byte) error {
	if err := e.wal.Write([]byte(key), value, false); err != nil {
		return fmt.Errorf("engine: put: wal write failed: %w", err)
	}

	if err := e.memtable.Put(key, value); err != nil {
		return fmt.Errorf("engine: put: memtable put failed: %w", err)
	}

	if e.memtable.IsFull() {
		if err := e.Flush(); err != nil {
			return fmt.Errorf("engine: put: flush failed: %w", err)
		}
	}
	return nil
}
