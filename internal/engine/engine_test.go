package engine

import "testing"

func TestEnginePutWritesToWALBeforeMemtable(t *testing.T) {
	wal := &mockWAL{}
	mt := newMockMemtable()
	e := New(wal, mt, &mockSSTableWriter{}, &mockSSTableReader{})

	if err := e.Put("key1", []byte("value1")); err != nil {
		t.Fatal(err)
	}

	if len(wal.writes) != 1 {
		t.Fatalf("expected 1 WAL write, got %d", len(wal.writes))
	}
	if string(wal.writes[0].key) != "key1" || wal.writes[0].tombstone {
		t.Fatal("unexpected WAL write contents")
	}
	if string(wal.writes[0].value) != "value1" {
		t.Fatal("unexpected WAL write value")
	}
}

func TestEnginePutStoresInMemtable(t *testing.T) {
	mt := newMockMemtable()
	e := New(&mockWAL{}, mt, &mockSSTableWriter{}, &mockSSTableReader{})

	if err := e.Put("key1", []byte("value1")); err != nil {
		t.Fatal(err)
	}

	val, found, tombstone := mt.Get("key1")
	if !found || tombstone {
		t.Fatal("expected key1 to be present and not a tombstone")
	}
	if string(val) != "value1" {
		t.Fatalf("expected value1, got %s", val)
	}
}

func TestEnginePutTriggersFlushWhenFull(t *testing.T) {
	writer := &mockSSTableWriter{}
	mt := newMockMemtable()
	mt.full = true // simulate an already-full memtable for this test

	e := New(&mockWAL{}, mt, writer, &mockSSTableReader{})

	if err := e.Put("key1", []byte("value1")); err != nil {
		t.Fatal(err)
	}

	if !writer.writeCalled {
		t.Fatal("expected Flush to call SSTableWriter.Write when memtable is full")
	}
}

func TestEngineDeleteWritesTombstoneToWAL(t *testing.T) {
	wal := &mockWAL{}
	e := New(wal, newMockMemtable(), &mockSSTableWriter{}, &mockSSTableReader{})

	if err := e.Delete("key1"); err != nil {
		t.Fatal(err)
	}

	if len(wal.writes) != 1 || !wal.writes[0].tombstone {
		t.Fatal("expected a tombstone write to the WAL")
	}
}

func TestEngineDeleteMarksMemtableTombstone(t *testing.T) {
	mt := newMockMemtable()
	e := New(&mockWAL{}, mt, &mockSSTableWriter{}, &mockSSTableReader{})

	e.Put("key1", []byte("value1"))
	if err := e.Delete("key1"); err != nil {
		t.Fatal(err)
	}

	_, found, tombstone := mt.Get("key1")
	if !found || !tombstone {
		t.Fatal("expected key1 to be marked as a tombstone after delete")
	}
}

func TestEngineDeleteTriggersFlushWhenFull(t *testing.T) {
	writer := &mockSSTableWriter{}
	mt := newMockMemtable()
	mt.full = true

	e := New(&mockWAL{}, mt, writer, &mockSSTableReader{})

	if err := e.Delete("key1"); err != nil {
		t.Fatal(err)
	}

	if !writer.writeCalled {
		t.Fatal("expected Flush to call SSTableWriter.Write when memtable is full")
	}
}

func TestEngineGetFromMemtable(t *testing.T) {
	mt := newMockMemtable()
	e := New(&mockWAL{}, mt, &mockSSTableWriter{}, &mockSSTableReader{})

	e.Put("key1", []byte("value1"))

	val, found := e.Get("key1")
	if !found {
		t.Fatal("expected to find key1 in the memtable")
	}

	if string(val) != "value1" {
		t.Fatalf("expected value1, got %s", val)
	}
}

func TestEngineGetHidesTombstonedKey(t *testing.T) {
	mt := newMockMemtable()
	e := New(&mockWAL{}, mt, &mockSSTableWriter{}, &mockSSTableReader{})

	e.Put("key1", []byte("value1"))
	e.Delete("key1")

	_, found := e.Get("key1")
	if found {
		t.Fatal("expected a deleted key to be reported as not found")
	}
}

func TestEngineGetFallsBackToSSTable(t *testing.T) {
	reader := &mockSSTableReader{data: map[string][]byte{"key1": []byte("from-sstable")}}
	e := New(&mockWAL{}, newMockMemtable(), &mockSSTableWriter{}, reader)

	val, found := e.Get("key1")
	if !found {
		t.Fatal("expected to find key1 in the SSTable")
	}
	if string(val) != "from-sstable" {
		t.Fatalf("expected value from SSTable, got %s", val)
	}
}

func TestEngineGetNotFoundAnywhere(t *testing.T) {
	e := New(&mockWAL{}, newMockMemtable(), &mockSSTableWriter{}, &mockSSTableReader{})

	_, found := e.Get("missing")
	if found {
		t.Fatal("expected a key that exists nowhere to be reported as not found")
	}
}
