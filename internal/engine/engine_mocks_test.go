package engine

import (
	"github.com/Dzekanaa/KevValueEngine/internal/memtable"
	"github.com/Dzekanaa/KevValueEngine/internal/wal"
)

// mockWAL is an in-memory WAL used to test Engine logic without a real
// WAL, tracking every write it receives.
type mockWAL struct {
	writes         []mockWALWrite
	recoverRecords []*wal.Record
}

type mockWALWrite struct {
	key       []byte
	value     []byte
	tombstone bool
}

func (m *mockWAL) Write(key []byte, value []byte, tombstone bool) error {
	m.writes = append(m.writes, mockWALWrite{key, value, tombstone})
	return nil
}

func (m *mockWAL) Recover() ([]*wal.Record, error) { return m.recoverRecords, nil }

func (m *mockWAL) ReadAll() ([]*wal.Record, error) { return nil, nil }

func (m *mockWAL) Cleanup() error { return nil }

// mockMemtable is an in-memory Memtable used to test Engine logic.
type mockMemtable struct {
	data map[string]mockEntry
	full bool
}

type mockEntry struct {
	value     []byte
	tombstone bool
}

func newMockMemtable() *mockMemtable {
	return &mockMemtable{data: make(map[string]mockEntry)}
}

func (m *mockMemtable) Put(key string, value []byte) error {
	m.data[key] = mockEntry{value: value, tombstone: false}
	return nil
}

func (m *mockMemtable) Delete(key string) error {
	m.data[key] = mockEntry{value: nil, tombstone: true}
	return nil
}

func (m *mockMemtable) Get(key string) (value []byte, found bool, tombstone bool) {
	entry, ok := m.data[key]
	if !ok {
		return nil, false, false
	}
	return entry.value, true, entry.tombstone
}

func (m *mockMemtable) GetSorted() []memtable.SortedEntry {
	entries := make([]memtable.SortedEntry, 0, len(m.data))
	for k, e := range m.data {
		entries = append(entries, memtable.SortedEntry{
			Key:       k,
			Value:     e.value,
			Tombstone: e.tombstone,
		})
	}
	return entries
}

func (m *mockMemtable) Size() int    { return len(m.data) }
func (m *mockMemtable) IsFull() bool { return m.full }
func (m *mockMemtable) Clear()       { m.data = make(map[string]mockEntry) }

// mockSSTableWriter records what it was asked to write, without
// touching disk.
type mockSSTableWriter struct {
	writeCalled bool
}

func (m *mockSSTableWriter) Write(entries []memtable.SortedEntry) (string, error) {
	m.writeCalled = true
	return "mock-sstable-id", nil
}

// mockSSTableReader is an in-memory stand-in for the SSTable read path.
// Unused until Get is implemented, but required to satisfy Engine's
// constructor
type mockSSTableReader struct {
	data map[string][]byte
}

func (m *mockSSTableReader) Get(key string) ([]byte, bool) {
	v, ok := m.data[key]
	return v, ok
}
