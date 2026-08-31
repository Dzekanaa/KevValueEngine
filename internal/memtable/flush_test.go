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

func TestRecoverFromWAL_PopulatesMemtable(t *testing.T) {
	mt := New(100)

	records := []WALRecord{
		{Key: []byte("key1"), Value: []byte("value1"), Tombstone: false},
		{Key: []byte("key2"), Value: []byte("value2"), Tombstone: false},
		{Key: []byte("key3"), Value: []byte("value3"), Tombstone: true}, // deleted
	}

	err := mt.RecoverFromWAL(records)
	if err != nil {
		t.Fatalf("RecoverFromWAL failed: %v", err)
	}

	if mt.Size() != 3 {
		t.Fatalf("expected 3 entries after recovery, got %d", mt.Size())
	}

	// Check normal entries
	value, found, tombstone := mt.Get("key1")
	if !found {
		t.Fatal("expected to find key1")
	}
	if tombstone {
		t.Fatal("expected key1 to not be tombstoned")
	}
	if string(value) != "value1" {
		t.Errorf("expected value1, got %s", string(value))
	}

	// Check tombstone
	_, found, tombstone = mt.Get("key3")
	if !found {
		t.Fatal("expected to find key3 with tombstone")
	}
	if !tombstone {
		t.Fatal("expected key3 to be tombstoned")
	}
}

func TestRecoverFromWAL_UpdatesExistingEntries(t *testing.T) {
	mt := New(100)

	// Pre-populate with an old value
	mt.Put("key1", []byte("old_value"))

	// Recover with updated value for same key
	records := []WALRecord{
		{Key: []byte("key1"), Value: []byte("new_value"), Tombstone: false},
	}

	err := mt.RecoverFromWAL(records)
	if err != nil {
		t.Fatalf("RecoverFromWAL failed: %v", err)
	}

	value, found, _ := mt.Get("key1")
	if !found {
		t.Fatal("expected to find key1")
	}

	if string(value) != "new_value" {
		t.Errorf("expected new_value, got %s", string(value))
	}
}

func TestRecoverFromWAL_FailsWhenCapacityExceeded(t *testing.T) {
	mt := New(2) // Very small capacity

	// Try to recover more than capacity
	records := []WALRecord{
		{Key: []byte("key1"), Value: []byte("value1"), Tombstone: false},
		{Key: []byte("key2"), Value: []byte("value2"), Tombstone: false},
		{Key: []byte("key3"), Value: []byte("value3"), Tombstone: false}, // This exceeds capacity
	}

	err := mt.RecoverFromWAL(records)

	if err == nil {
		t.Fatal("expected error when recovery exceeds capacity, got nil")
	}

	if mt.Size() != 2 {
		t.Fatalf("expected 2 entries before capacity error, got %d", mt.Size())
	}
}

func TestRecoverFromWAL_AllowsUpdateWhenAtCapacity(t *testing.T) {
	mt := New(2)

	// Pre-populate to capacity
	mt.Put("key1", []byte("value1"))
	mt.Put("key2", []byte("value2"))

	// Recovery updates existing key (should work even at capacity)
	records := []WALRecord{
		{Key: []byte("key1"), Value: []byte("updated"), Tombstone: false},
	}

	err := mt.RecoverFromWAL(records)
	if err != nil {
		t.Fatalf("RecoverFromWAL update at capacity failed: %v", err)
	}

	value, _, _ := mt.Get("key1")
	if string(value) != "updated" {
		t.Errorf("expected updated, got %s", string(value))
	}
}

func TestIntegration_FillAndFlush(t *testing.T) {
	mt := New(3) // Small capacity

	// Fill to capacity
	mt.Put("key1", []byte("value1"))
	mt.Put("key2", []byte("value2"))
	mt.Put("key3", []byte("value3"))

	if !mt.IsFull() {
		t.Fatal("expected memtable to be full after 3 inserts")
	}

	// Flush
	writer := &MockSSTableWriter{WriteID: "sstable_001"}
	_, err := mt.Flush(writer)

	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Verify memtable is cleared
	if mt.Size() != 0 {
		t.Fatalf("expected empty memtable after flush, got %d entries", mt.Size())
	}

	if mt.IsFull() {
		t.Fatal("expected memtable to not be full after flush")
	}
}
