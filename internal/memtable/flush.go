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

// Flush persists all currecnt entries to an SSTable and then clears the memtable.
// It takes sorted entries, writes them via the provided SSTableWriter,
// and clears the memtable after successful write.
func (m *Memtable) Flush(writer SSTableWriter) (string, error) {
	// Get all entries sorted by key
	entries := m.GetSorted()

	if len(entries) == 0 {
		return "", fmt.Errorf("cannot flush empty memtable")
	}
}
