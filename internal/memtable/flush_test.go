package memtable

import (
	"fmt"
	"testing"
)

// MockSSTableWriter is a test double for SSTableWriter
type MockSSTableWriter struct {
	WriteCalled  bool
	WriteError   error
	WriteID      string
	WriteEntries []SortedEntry
}

func (m *MockSSTableWriter) Write(entries []SortedEntry) (id string, err error) {
	m.WriteCalled = true
	m.WriteEntries = make([]SortedEntry, len(entries))
	copy(m.WriteEntries, entries)

	if m.WriteError != nil {
		return "", m.WriteError
	}

	return m.WriteID, nil
}

func TestFlush_PersistsAndClearsMemtable(t *testing.T) {
	mt := New(100)

	mt.Put("key1", []byte("value1"))
	mt.Put("key2", []byte("value2"))

	if mt.Size() != 2 {
		t.Fatalf("expected 2 entries before flush, got %d", mt.Size())
	}

	writer := &MockSSTableWriter{WriteID: "sstable_001"}
	id, err := mt.Flush(writer)

	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if id != "sstable_001" {
		t.Errorf("expected ID 'sstable_001', got %s", id)
	}

	if !writer.WriteCalled {
		t.Fatal("expected SSTableWriter.Write to be called")
	}

	if len(writer.WriteEntries) != 2 {
		t.Fatalf("expected 2 entries written, got %d", len(writer.WriteEntries))
	}

	if mt.Size() != 0 {
		t.Fatalf("expected memtable to be empty after flush, got %d entries", mt.Size())
	}
}

func TestFlush_SortsEntriesBeforeWrite(t *testing.T) {
	mt := New(100)

	mt.Put("zebra", []byte("z_value"))
	mt.Put("apple", []byte("a_value"))
	mt.Put("banana", []byte("b_value"))

	writer := &MockSSTableWriter{WriteID: "sstable_001"}
	_, err := mt.Flush(writer)

	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if len(writer.WriteEntries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(writer.WriteEntries))
	}

	// Verify entries are sorted
	if writer.WriteEntries[0].Key != "apple" {
		t.Errorf("expected first key 'apple', got %s", writer.WriteEntries[0].Key)
	}
	if writer.WriteEntries[1].Key != "banana" {
		t.Errorf("expected second key 'banana', got %s", writer.WriteEntries[1].Key)
	}
	if writer.WriteEntries[2].Key != "zebra" {
		t.Errorf("expected third key 'zebra', got %s", writer.WriteEntries[2].Key)
	}
}

func TestFlush_IncludesTombstones(t *testing.T) {
	mt := New(100)

	mt.Put("key1", []byte("value1"))
	mt.Delete("key2") // tombstone

	writer := &MockSSTableWriter{WriteID: "sstable_001"}
	_, err := mt.Flush(writer)

	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if len(writer.WriteEntries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(writer.WriteEntries))
	}

	// Check for tombstone
	tombstoneFound := false
	for _, entry := range writer.WriteEntries {
		if entry.Tombstone {
			tombstoneFound = true
			break
		}
	}

	if !tombstoneFound {
		t.Fatal("expected tombstone marker in flushed entries")
	}
}

func TestFlush_FailsOnEmptyMemtable(t *testing.T) {
	mt := New(100)

	writer := &MockSSTableWriter{WriteID: "sstable_001"}
	_, err := mt.Flush(writer)

	if err == nil {
		t.Fatal("expected error when flushing empty memtable, got nil")
	}
}

func TestFlush_PropagatesWriterError(t *testing.T) {
	mt := New(100)

	mt.Put("key1", []byte("value1"))

	writer := &MockSSTableWriter{WriteError: fmt.Errorf("disk write failed")}
	_, err := mt.Flush(writer)

	if err == nil {
		t.Fatal("expected error from failed write, got nil")
	}

	// Memtable should NOT be cleared if flush fails
	if mt.Size() != 1 {
		t.Errorf("expected memtable to retain entries on flush error, got %d entries", mt.Size())
	}
}

func TestClear_RemovesAllEntries(t *testing.T) {
	mt := New(100)

	mt.Put("key1", []byte("value1"))
	mt.Put("key2", []byte("value2"))

	if mt.Size() != 2 {
		t.Fatalf("expected 2 entries before clear, got %d", mt.Size())
	}

	mt.Clear()

	if mt.Size() != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", mt.Size())
	}

	_, found, _ := mt.Get("key1")
	if found {
		t.Fatal("expected key1 to be gone after clear")
	}
}

func TestClear_AllowsNewInsertionsAfterClear(t *testing.T) {
	mt := New(100)

	mt.Put("key1", []byte("value1"))
	mt.Clear()

	err := mt.Put("key2", []byte("value2"))
	if err != nil {
		t.Fatalf("Put after clear failed: %v", err)
	}

	value, found, _ := mt.Get("key2")
	if !found {
		t.Fatal("expected to find key2 after clear")
	}

	if string(value) != "value2" {
		t.Errorf("expected value2, got %s", string(value))
	}
}
