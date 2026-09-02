package memtable

import (
	"fmt"
	"testing"
)

func TestPut_AddsSingleEntry(t *testing.T) {
	mt := New(100)

	err := mt.Put("key1", []byte("value1"))
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	value, found, tombstone := mt.Get("key1")
	if !found {
		t.Fatal("expected to find key1")
	}
	if tombstone {
		t.Fatal("expected key1 to not be tombstoned (marked as deleted)")
	}
	if string(value) != "value1" {
		t.Errorf("expected value1, got %s", string(value))
	}
}

func TestPut_UpdatesExistingEntry(t *testing.T) {
	mt := New(100)

	mt.Put("key1", []byte("value1"))
	mt.Put("key1", []byte("value2")) // update

	value, found, _ := mt.Get("key1")
	if !found {
		t.Fatal("expected to find key1")
	}
	if string(value) != "value2" {
		t.Errorf("expected value2, got %s", string(value))
	}
}

func TestPut_RejectWhenFull(t *testing.T) {
	mt := New(2)

	mt.Put("key1", []byte("value1"))
	mt.Put("key2", []byte("value2"))

	// Memtable is now full (2/2)
	err := mt.Put("key3", []byte("value3"))
	if err == nil {
		t.Fatal("expected error when putting into full memtable, got nil")
	}

	if !mt.IsFull() {
		t.Fatal("expected memtable to be marked full")
	}
}

func TestPut_AllowsUpdateWhenFull(t *testing.T) {
	mt := New(2)

	mt.Put("key1", []byte("value1"))
	mt.Put("key2", []byte("value2"))

	// Updating an existing key should work even when full
	err := mt.Put("key1", []byte("updated"))
	if err != nil {
		t.Fatalf("Put update should work when full, got error: %v", err)
	}

	value, _, _ := mt.Get("key1")
	if string(value) != "updated" {
		t.Errorf("expected updated, got %s", string(value))
	}
}

func TestDelete_MarksTombstone(t *testing.T) {
	mt := New(100)

	mt.Put("key1", []byte("value1"))
	err := mt.Delete("key1")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, found, tombstone := mt.Get("key1")
	if !found {
		t.Fatal("expected to find key1 (with tombstone)")
	}
	if !tombstone {
		t.Fatal("expected key1 to be tombstoned")
	}
}

func TestDelete_CreatesTombstoneForNonExistentKey(t *testing.T) {
	mt := New(100)

	err := mt.Delete("nonexistent")
	if err != nil {
		t.Fatalf("Delete on nonexistent key failed: %v", err)
	}

	_, found, tombstone := mt.Get("nonexistent")
	if !found {
		t.Fatal("expected to find nonexistent (with tombstone)")
	}
	if !tombstone {
		t.Fatal("expected nonexistent to be tombstoned")
	}
}

func TestGet_NonExistentKeyReturnsFalse(t *testing.T) {
	mt := New(100)

	_, found, _ := mt.Get("does_not_exist")
	if found {
		t.Fatal("expected Get to return found=false for nonexistent key")
	}
}

func TestSize_ReturnsNumberOfEntries(t *testing.T) {
	mt := New(100)

	if mt.Size() != 0 {
		t.Fatalf("expected size 0, got %d", mt.Size())
	}

	mt.Put("key1", []byte("value1"))
	if mt.Size() != 1 {
		t.Fatalf("expected size 1, got %d", mt.Size())
	}

	mt.Put("key2", []byte("value2"))
	if mt.Size() != 2 {
		t.Fatalf("expected size 2, got %d", mt.Size())
	}
}

func TestGetMaxSize_ReturnsConfiguredSize(t *testing.T) {
	mt := New(500)

	if mt.GetMaxSize() != 500 {
		t.Errorf("expected MaxSize 500, got %d", mt.GetMaxSize())
	}
}

func TestThreadSafety_ConcurrentReads(t *testing.T) {
	mt := New(1000)

	mt.Put("key1", []byte("value1"))

	// Prepares a "done" channel (checklist) to track when threads finish
	// Starts 10 background threads which call mt.Get("key1")
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, _, _ = mt.Get("key1")
			done <- true
		}()
	}

	// Waits for the 10 threads to send their "true" signals
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestThreadSafety_ConcurrentWrites(t *testing.T) {
	mt := New(1000)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		idx := i // current loop index for the thread
		go func() {
			key := fmt.Sprintf("key%d", idx) // key0, key1, ..., key9
			value := []byte(fmt.Sprintf("value%d", idx))
			mt.Put(key, value)
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if mt.Size() != 10 {
		t.Fatalf("expected 10 entries after concurrent writes, got %d", mt.Size())
	}
}

func TestGetSorted_ReturnsSortedEntries(t *testing.T) {
	mt := New(100)

	mt.Put("zebra", []byte("z_value"))
	mt.Put("apple", []byte("a_value"))
	mt.Put("banana", []byte("b_value"))

	sorted := mt.GetSorted()

	if len(sorted) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(sorted))
	}

	if sorted[0].Key != "apple" {
		t.Errorf("expected first key 'apple', got %s", sorted[0].Key)
	}
	if sorted[1].Key != "banana" {
		t.Errorf("expected second key 'banana', got %s", sorted[1].Key)
	}
	if sorted[2].Key != "zebra" {
		t.Errorf("expected third key 'zebra', got %s", sorted[2].Key)
	}
}

func TestNew_UsesMaxSizeConstantWhenZeroOrLower(t *testing.T) {
	mt := New(0)

	if mt.GetMaxSize() != MaxSize {
		t.Errorf("expected maxSize to be %d, got %d", MaxSize, mt.GetMaxSize())
	}
}

func TestClear_RemovesAllEntries(t *testing.T) {
	mt := New(100)
	mt.Put("key1", []byte("value1"))
	mt.Put("key2", []byte("value2"))

	mt.Clear()

	if mt.Size() != 0 {
		t.Fatalf("expected 0 entries after clear, got %d", mt.Size())
	}
}
