package wal

import (
	"os"
	"testing"

	"github.com/Dzekanaa/KevValueEngine/internal/blockmanager"
)

func TestWALWithRealBlockManager(t *testing.T) {
	dir, err := os.MkdirTemp("", "wal_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(dir)

	bm, err := blockmanager.New(4096, 10) // block size must be a multiple of 4096
	if err != nil {
		t.Fatalf("failed to create block manager: %v", err)
	}

	wal := NewWAL(dir, bm, 4096, 2) // recordsPerSegment = 2

	if err := wal.Write([]byte("key1"), []byte("value1"), false); err != nil {
		t.Fatal(err)
	}
	if err := wal.Write([]byte("key2"), []byte("value2"), false); err != nil {
		t.Fatal(err)
	}
	if err := wal.Write([]byte("key3"), nil, true); err != nil {
		t.Fatal(err)
	}

	// simulate a restart against the same on-disk directory
	wal2 := NewWAL(dir, bm, 4096, 2)
	records, err := wal2.Recover()
	if err != nil {
		t.Fatal(err)
	}

	if len(records) != 3 {
		t.Fatalf("expected 3 records, got %d", len(records))
	}
	if string(records[0].Key) != "key1" {
		t.Fatalf("expected key1, got %s", records[0].Key)
	}
	if !records[2].Tombstone {
		t.Fatal("expected record 2 to be a tombstone")
	}

	// write after recovery must not overwrite
	if err := wal2.Write([]byte("key4"), []byte("value4"), false); err != nil {
		t.Fatal(err)
	}
	all, err := wal2.ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 records after write-after-recovery, got %d", len(all))
	}
}
