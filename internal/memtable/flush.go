package memtable

import "fmt"

// SSTableWriter is the interface for persisting sorted entries to disk.
type SSTableWriter interface {
	Write(entries []SortedEntry) (id string, err error)
}

// WALRecord represents a record from the Write-Ahead Log used during recovery.
type WALRecord struct {
	Key       []byte
	Value     []byte
	Tombstone bool
}

// Flush persists all current entries to an SSTable and then clears the memtable.
// It takes sorted entries, writes them via the provided SSTableWriter,
// and clears the memtable after successful write.
func (m *Memtable) Flush(writer SSTableWriter) (string, error) {
	// Get all entries sorted by key
	entries := m.GetSorted()

	if len(entries) == 0 {
		return "", fmt.Errorf("cannot flush empty memtable")
	}

	// Write to SSTable
	id, err := writer.Write(entries)
	if err != nil {
		return "", fmt.Errorf("failed to write SSTable: %w", err)
	}

	// Clear memtable after successful flush
	m.Clear()

	return id, nil
}

// Clear empties the memtable, discarding all current entries.
// Called by the engine right after entries have been durably persisted
// to an SSTable during a flush.
func (m *Memtable) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data = make(map[string]*Entry, m.maxSize)
}

// RecoverFromWAL loads all records from WAL and fills the memtable.
// This is called during system startup to restore the most recent state
// from the write-ahead log before the last crash.
func (m *Memtable) RecoverFromWAL(records []WALRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, record := range records {
		key := string(record.Key)

		// Check if we have capacity (unless updating existing key)
		_, exists := m.data[key]
		if !exists && len(m.data) >= m.maxSize {
			return fmt.Errorf("memtable capacity exceeded during recovery: %d / %d entries",
				len(m.data), m.maxSize)
		}

		// Add or update the entry
		m.data[key] = &Entry{
			Value:     record.Value,
			Tombstone: record.Tombstone,
		}
	}

	return nil
}
