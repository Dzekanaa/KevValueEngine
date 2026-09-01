package sstable

import (
	"errors"
	"testing"

	"github.com/Dzekanaa/KevValueEngine/internal/memtable"
)

// mockBlockManager implements BlockManager in memory.
type mockBlockManager struct {
	blocks map[string]map[int][]byte // path -> blockNum -> data
}

func newMockBlockManager() *mockBlockManager {
	return &mockBlockManager{
		blocks: make(map[string]map[int][]byte),
	}
}

func (m *mockBlockManager) ReadBlock(path string, blockNum int) ([]byte, error) {
	fileBlocks, ok := m.blocks[path]
	if !ok {
		return nil, errors.New("file not found")
	}
	data, ok := fileBlocks[blockNum]
	if !ok {
		return nil, errors.New("block not found")
	}
	return data, nil
}

func (m *mockBlockManager) WriteBlock(path string, blockNum int, data []byte) error {
	if m.blocks[path] == nil {
		m.blocks[path] = make(map[int][]byte)
	}
	m.blocks[path][blockNum] = data
	return nil
}

// makeSortedEntries creates a slice of SortedEntry from key-value pairs.
func makeSortedEntries(pairs ...struct {
	key       string
	value     string
	tombstone bool
},
) []memtable.SortedEntry {
	entries := make([]memtable.SortedEntry, len(pairs))
	for i, p := range pairs {
		entries[i] = memtable.SortedEntry{
			Key:       p.key,
			Value:     []byte(p.value),
			Tombstone: p.tombstone,
		}
	}
	return entries
}

// TestWriteData tests serialization and block writing of Data.
func TestWriteData(t *testing.T) {
	bm := newMockBlockManager()
	path := "test.data"
	blockSize := 64
	entries := makeSortedEntries(
		struct {
			key, value string
			tombstone  bool
		}{"a", "apple", false},
		struct {
			key, value string
			tombstone  bool
		}{"b", "banana", false},
	)

	index, err := writeData(bm, path, blockSize, entries)
	if err != nil {
		t.Fatalf("writeData failed: %v", err)
	}

	if len(index) != 2 {
		t.Fatalf("expected 2 index entries, got %d", len(index))
	}
	if index[0].Key != "a" || index[1].Key != "b" {
		t.Errorf("index keys mismatch")
	}

	// Check that blocks were written
	if _, ok := bm.blocks[path]; !ok {
		t.Fatal("no blocks written")
	}

	// Read back first record
	tombstone, value, err := readDataRecord(bm, path, index[0].BlockNum, index[0].Offset)
	if err != nil {
		t.Fatalf("readDataRecord failed: %v", err)
	}
	if tombstone {
		t.Error("expected tombstone false")
	}
	if string(value) != "apple" {
		t.Errorf("expected 'apple', got '%s'", value)
	}
}

// TestReadDataRecordOutOfBounds tests error handling.
func TestReadDataRecordOutOfBounds(t *testing.T) {
	bm := newMockBlockManager()
	path := "test.data"
	blockSize := 64
	// Write a block with one record
	entries := makeSortedEntries(struct {
		key, value string
		tombstone  bool
	}{"x", "xyz", false})
	index, err := writeData(bm, path, blockSize, entries)
	if err != nil {
		t.Fatal(err)
	}
	// Try reading at an invalid offset
	_, _, err = readDataRecord(bm, path, index[0].BlockNum, 9999)
	if err == nil {
		t.Error("expected error for out-of-bounds offset")
	}
}

// TestWriteIndex tests Index writing and reading.
func TestWriteIndex(t *testing.T) {
	bm := newMockBlockManager()
	path := "test.idx"
	blockSize := 64
	entries := []indexEntry{
		{Key: "a", BlockNum: 0, Offset: 0},
		{Key: "b", BlockNum: 1, Offset: 10},
		{Key: "c", BlockNum: 2, Offset: 20},
	}

	err := writeIndex(bm, path, blockSize, entries)
	if err != nil {
		t.Fatalf("writeIndex failed: %v", err)
	}

	// Read all entries and verify
	readEntries, err := readAllIndexEntries(bm, path)
	if err != nil {
		t.Fatalf("readAllIndexEntries failed: %v", err)
	}
	if len(readEntries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(readEntries))
	}
	for i, e := range entries {
		if readEntries[i].Key != e.Key || readEntries[i].BlockNum != e.BlockNum || readEntries[i].Offset != e.Offset {
			t.Errorf("entry %d mismatch", i)
		}
	}

	// Test binary search
	blockNum, offset, found, err := readIndex(bm, path, "b")
	if err != nil {
		t.Fatalf("readIndex failed: %v", err)
	}
	if !found {
		t.Error("key 'b' should be found")
	}
	if blockNum != 1 || offset != 10 {
		t.Errorf("expected (1,10), got (%d,%d)", blockNum, offset)
	}

	// Test not found
	_, _, found, _ = readIndex(bm, path, "z")
	if found {
		t.Error("key 'z' should not be found")
	}
}

// TestWriteBlob tests blob serialization with length prefix.
func TestWriteBlob(t *testing.T) {
	bm := newMockBlockManager()
	path := "test.blob"
	blockSize := 32
	data := []byte("hello world")

	err := writeBlob(bm, path, blockSize, data)
	if err != nil {
		t.Fatalf("writeBlob failed: %v", err)
	}

	read, err := readBlob(bm, path, blockSize)
	if err != nil {
		t.Fatalf("readBlob failed: %v", err)
	}
	if string(read) != "hello world" {
		t.Errorf("expected 'hello world', got '%s'", read)
	}
}

// TestSummary tests serialization and range checking.
func TestSummary(t *testing.T) {
	s := summary{MinKey: "b", MaxKey: "m"}
	buf := s.serialize()
	des := deserializeSummary(buf)
	if des.MinKey != "b" || des.MaxKey != "m" {
		t.Errorf("deserialization mismatch: got (%s,%s)", des.MinKey, des.MaxKey)
	}

	if !s.inRange("g") {
		t.Error("'g' should be in range")
	}
	if s.inRange("a") {
		t.Error("'a' should not be in range")
	}
	if s.inRange("z") {
		t.Error("'z' should not be in range")
	}
}

// TestManagerWriteAndGet tests the full Manager.
func TestManagerWriteAndGet(t *testing.T) {
	bm := newMockBlockManager()
	dir := "/fake"
	blockSize := 64
	mgr := NewManager(bm, dir, blockSize)

	entries := makeSortedEntries(
		struct {
			key, value string
			tombstone  bool
		}{"car", "toyota", false},
		struct {
			key, value string
			tombstone  bool
		}{"dog", "labrador", false},
	)
	id, err := mgr.Write(entries)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}
	if id == "" {
		t.Error("empty id returned")
	}

	// Get existing key
	val, found := mgr.Get("car")
	if !found {
		t.Error("key 'car' not found")
	}
	if string(val) != "toyota" {
		t.Errorf("expected 'toyota', got '%s'", val)
	}

	// Get non-existing key
	_, found = mgr.Get("elephant")
	if found {
		t.Error("key 'elephant' should not be found")
	}

	// Write another table with newer entries and test that newest is returned
	entries2 := makeSortedEntries(
		struct {
			key, value string
			tombstone  bool
		}{"car", "tesla", false}, // newer value
	)
	_, err = mgr.Write(entries2)
	if err != nil {
		t.Fatal(err)
	}
	val, found = mgr.Get("car")
	if !found {
		t.Error("key 'car' not found after second write")
	}
	if string(val) != "tesla" {
		t.Errorf("expected 'tesla' (newest), got '%s'", val)
	}

	// Test tombstone: delete key
	entries3 := makeSortedEntries(
		struct {
			key, value string
			tombstone  bool
		}{"car", "", true},
	)
	_, err = mgr.Write(entries3)
	if err != nil {
		t.Fatal(err)
	}
	_, found = mgr.Get("car")
	if found {
		t.Error("key 'car' should be deleted (tombstone)")
	}
}

// TestManagerGetWithFilter skips tables not containing key.
func TestManagerGetWithFilter(t *testing.T) {
	bm := newMockBlockManager()
	dir := "/fake"
	blockSize := 64
	mgr := NewManager(bm, dir, blockSize)

	// Write table with keys "apple", "apricot"
	entries := makeSortedEntries(
		struct {
			key, value string
			tombstone  bool
		}{"apple", "fruit", false},
		struct {
			key, value string
			tombstone  bool
		}{"apricot", "fruit", false},
	)
	_, err := mgr.Write(entries)
	if err != nil {
		t.Fatal(err)
	}

	// This key is not in the table; Bloom filter should say "no" and skip Index.
	// Since we can't easily mock the bloom filter, we rely on the actual implementation.
	_, found := mgr.Get("banana")
	if found {
		t.Error("'banana' should not be found")
	}
}

// TestManagerGetWithSummaryRange tests that summary range filtering works.
func TestManagerGetWithSummaryRange(t *testing.T) {
	bm := newMockBlockManager()
	dir := "/fake"
	blockSize := 64
	mgr := NewManager(bm, dir, blockSize)

	// Write table with keys "a" to "m"
	entries := makeSortedEntries(
		struct {
			key, value string
			tombstone  bool
		}{"a", "1", false},
		struct {
			key, value string
			tombstone  bool
		}{"m", "2", false},
	)
	_, err := mgr.Write(entries)
	if err != nil {
		t.Fatal(err)
	}

	// Key outside range ("z") should be skipped by summary before Index.
	_, found := mgr.Get("z")
	if found {
		t.Error("'z' should not be found")
	}
}

// TestManagerEmptyWrite ensures error on empty entry set.
func TestManagerEmptyWrite(t *testing.T) {
	bm := newMockBlockManager()
	dir := "/fake"
	blockSize := 64
	mgr := NewManager(bm, dir, blockSize)

	_, err := mgr.Write([]memtable.SortedEntry{})
	if err == nil {
		t.Error("expected error for empty entries")
	}
}
