// Package engine orchestrates the WAL, Memtable, and SSTable components
// into the system's PUT/GET/DELETE operations, including flush and
// startup recovery.
package engine

import (
	"github.com/Dzekanaa/KevValueEngine/internal/memtable"
	"github.com/Dzekanaa/KevValueEngine/internal/wal"
)

// ---------------------------------------------------------------------
// These are the contracts engine depends on. Each college implements
// the corresponding method(s) on their own concrete type; as long as
// the method signatures below match, everything wires together
// automatically without engine needing any changes.
// ---------------------------------------------------------------------

// WAL is the interface the engine uses for write-ahead logging.
// Implemented by the wal package.
type WAL interface {
	// Write appends a record to the log before it is applied to the
	// Memtable. Already implemented.
	Write(key []byte, value []byte, tombstone bool) error

	// ReadAll returns every record from every segment, oldest first.
	// Used by Recover. Already implemented.
	ReadAll() ([]*wal.Record, error)

	// Cleanup deletes every WAL segment older than the currently
	// active one. Must only be called after a flush has durably
	// persisted the Memtable to an SSTable — never before.
	// Already implemented.
	Cleanup() error
}

// WALRecord is the subset of a WAL record the engine needs to replay
// it into the Memtable during recovery.
type WALRecord struct {
	Key       []byte
	Value     []byte
	Tombstone bool
}

// Memtable is the interface the engine uses for the in-memory write
// buffer. Implemented by the memtable package.
type Memtable interface {
	// Put, Delete, Get, GetSorted, Size, IsFull are already implemented.
	Put(key string, value []byte) error
	Delete(key string) error
	Get(key string) (value []byte, found bool, tombstone bool)
	GetSorted() []memtable.SortedEntry
	Size() int
	IsFull() bool

	// Clear empties the memtable, discarding all current entries.
	// Called by the engine right after those entries have been
	// durably persisted to an SSTable during a flush.
	//
	// Niksa
	// TODO: implement this. Take the write lock and
	// reset the internal map, e.g. m.data = make(map[string]*Entry, m.maxSize).
	Clear()
}

// SSTableWriter is the interface the engine uses to persist a flushed
// Memtable to disk as a new SSTable. Implemented by the sstable package.
type SSTableWriter interface {
	// Write persists entries (already sorted by key, as returned by
	// Memtable.GetSorted) as a new SSTable and returns an identifier
	// for the created table.
	//
	// Dzektor
	// TODO: implement this. entries is guaranteed
	// non-empty and sorted by key when Write is called from Flush.
	Write(entries []memtable.SortedEntry) (id string, err error)
}

// SSTableReader is the interface the engine uses to look up a key
// across existing SSTables when it isn't found in the Memtable.
// Implemented by the sstable package.
type SSTableReader interface {
	// Get searches all known SSTables for key, newest first, using
	// each table's Bloom filter to skip tables that can't contain it,
	// then its Summary and Index to locate the value in Data.
	// found is false if the key isn't in any SSTable, or is present
	// only as a tombstone.
	//
	// Dzektor
	// TODO: implement this once Write, the Bloom
	// filter integration, Index, and Summary exist. Not needed yet —
	// Get() below has a TODO marking where this plugs in.
	Get(key string) (value []byte, found bool)
}
