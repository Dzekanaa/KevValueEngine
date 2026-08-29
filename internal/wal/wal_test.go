package wal

import (
	"os"
	"testing"

	"github.com/Dzekanaa/KevValueEngine/internal/blockmanager"
)

func TestWAL(t *testing.T) {
	dir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	bm, err := blockmanager.New(4096, 10)
	if err != nil {
		t.Fatalf("failed to create block manager: %v", err)
	}
	blockSize := 4096
	recordsPerSegment := 2 // small on purpose, to force a segment rollover

	wal := NewWAL(dir, bm, blockSize, recordsPerSegment)

	testRecords := []struct {
		key       string
		value     string
		tombstone bool
	}{
		{"key1", "value1", false},
		{"key2", "value2", false},
		{"key3", "value3", false},
		{"key4", "value4", false},
		{"key5", "", true},
	}

	for _, r := range testRecords {
		var val []byte
		if !r.tombstone {
			val = []byte(r.value)
		}
		if err := wal.Write([]byte(r.key), val, r.tombstone); err != nil {
			t.Fatal(err)
		}
	}

	if wal.currentSegment != 3 {
		t.Fatalf("expected currentSegment 3, got %d", wal.currentSegment)
	}

	// simulate a restart: a fresh WAL over the same bm and dir
	wal2 := NewWAL(dir, bm, blockSize, recordsPerSegment)

	records, err := wal2.ReadAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 5 {
		t.Fatalf("expected 5 records, got %d", len(records))
	}

	for i, r := range testRecords {
		if string(records[i].Key) != r.key {
			t.Fatalf("record %d: expected key %s, got %s", i, r.key, records[i].Key)
		}
		if records[i].Tombstone != r.tombstone {
			t.Fatalf("record %d: tombstone mismatch", i)
		}
	}
}

func TestDeleteSegmentGuard(t *testing.T) {
	dir, err := os.MkdirTemp("", "wal_test_delete")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	bm, err := blockmanager.New(4096, 10)
	if err != nil {
		t.Fatalf("failed to create block manager: %v", err)
	}

	wal := NewWAL(dir, bm, 4096, 2)

	wal.Write([]byte("k1"), []byte("v1"), false)
	wal.Write([]byte("k2"), []byte("v2"), false)
	wal.Write([]byte("k3"), []byte("v3"), false) // rolls over into segment 2

	// segment 2 is active — must be rejected regardless of disk state
	if err := wal.DeleteSegment(2); err == nil {
		t.Fatal("expected error when deleting the active segment, got nil")
	}

	// segment 1 is closed — should succeed now that real files exist on disk
	if err := wal.DeleteSegment(1); err != nil {
		t.Fatalf("expected successful delete of closed segment, got error: %v", err)
	}
}

func TestRecoverResumesWithoutOverwriting(t *testing.T) {
	dir, err := os.MkdirTemp("", "wal_test_recover")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	bm, err := blockmanager.New(4096, 10)
	if err != nil {
		t.Fatalf("failed to create block manager: %v", err)
	}
	blockSize := 4096
	recordsPerSegment := 10

	wal := NewWAL(dir, bm, blockSize, recordsPerSegment)
	wal.Write([]byte("k1"), []byte("v1"), false)
	wal.Write([]byte("k2"), []byte("v2"), false)

	wal2 := NewWAL(dir, bm, blockSize, recordsPerSegment)
	recovered, err := wal2.Recover()
	if err != nil {
		t.Fatal(err)
	}
	if len(recovered) != 2 {
		t.Fatalf("expected 2 recovered records, got %d", len(recovered))
	}

	wal2.Write([]byte("k3"), []byte("v3"), false)

	all, err := wal2.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 records after write-after-recovery, got %d", len(all))
	}
	if string(all[0].Key) != "k1" || string(all[1].Key) != "k2" || string(all[2].Key) != "k3" {
		t.Fatal("records out of order or overwritten")
	}
}
