package blockmanager

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteAndReadBlock_ReturnsExactData(t *testing.T) {
	bm := newTestBM(t, 4096, 10)
	path := testPath(t)

	data := []byte("hello world")
	if err := bm.WriteBlock(path, 0, data); err != nil {
		t.Fatalf("WriteBlock failed: %v", err)
	}

	got, err := bm.ReadBlock(path, 0)
	if err != nil {
		t.Fatalf("ReadBlock failed: %v", err)
	}

	if !bytes.HasPrefix(got, data) {
		t.Errorf("expected block to start with %q, got %q", data, got[:len(data)])
	}
}

func TestWriteBlock_PadsRemainderWithZeros(t *testing.T) {
	bm := newTestBM(t, 4096, 10)
	path := testPath(t)

	data := []byte("short")
	if err := bm.WriteBlock(path, 0, data); err != nil {
		t.Fatalf("WriteBlock failed: %v", err)
	}

	got, err := bm.ReadBlock(path, 0)
	if err != nil {
		t.Fatalf("ReadBlock failed: %v", err)
	}

	if len(got) != 4096 {
		t.Fatalf("expected block length 4096, got %d", len(got))
	}

	for i, b := range got[len(data):] {
		if b != 0 {
			t.Fatalf("expected padding to be zero at index %d, got %d", len(data)+i, b)
		}
	}
}

func TestWriteBlock_DataExceedsBlockSize_ReturnsError(t *testing.T) {
	bm := newTestBM(t, 4096, 10)
	path := testPath(t)

	data := make([]byte, 5000)
	err := bm.WriteBlock(path, 0, data)
	if err == nil {
		t.Fatal("expected error when data exceeds block size, got nil")
	}
}

func TestReadBlock_FileDoesNotExist_ReturnsErrBlockNotFound(t *testing.T) {
	bm := newTestBM(t, 4096, 10)
	path := filepath.Join(t.TempDir(), "does_not_exist.bin")

	_, err := bm.ReadBlock(path, 0)
	if err != ErrBlockNotFound {
		t.Fatalf("expected ErrBlockNotFound, got %v", err)
	}
}

func TestReadBlock_BlockBeyondEndOfFile_ReturnsErrBlockNotFound(t *testing.T) {
	bm := newTestBM(t, 4096, 10)
	path := testPath(t)

	if err := bm.WriteBlock(path, 0, []byte("only block 0")); err != nil {
		t.Fatalf("WriteBlock failed: %v", err)
	}

	_, err := bm.ReadBlock(path, 5)
	if err != ErrBlockNotFound {
		t.Fatalf("expected ErrBlockNotFound for block beyond EOF, got %v", err)
	}
}

func TestWriteBlock_MultipleBlocksAtDifferentOffsets(t *testing.T) {
	bm := newTestBM(t, 4096, 10)
	path := testPath(t)

	block0 := []byte("first block")
	block1 := []byte("second block")
	block2 := []byte("third block")

	if err := bm.WriteBlock(path, 0, block0); err != nil {
		t.Fatalf("WriteBlock(0) failed: %v", err)
	}
	if err := bm.WriteBlock(path, 1, block1); err != nil {
		t.Fatalf("WriteBlock(1) failed: %v", err)
	}
	if err := bm.WriteBlock(path, 2, block2); err != nil {
		t.Fatalf("WriteBlock(2) failed: %v", err)
	}

	for i, want := range [][]byte{block0, block1, block2} {
		got, err := bm.ReadBlock(path, i)
		if err != nil {
			t.Fatalf("ReadBlock(%d) failed: %v", i, err)
		}
		if !bytes.HasPrefix(got, want) {
			t.Errorf("block %d: expected prefix %q, got %q", i, want, got[:len(want)])
		}
	}
}

func TestReadBlock_UsesCacheWithoutRereadingDisk(t *testing.T) {
	bm := newTestBM(t, 4096, 10)
	path := testPath(t)

	original := []byte("cached value")
	if err := bm.WriteBlock(path, 0, original); err != nil {
		t.Fatalf("WriteBlock failed: %v", err)
	}

	// prime the cache
	if _, err := bm.ReadBlock(path, 0); err != nil {
		t.Fatalf("ReadBlock failed: %v", err)
	}

	// corrupt the file directly on disk, bypassing the block manager
	if err := os.WriteFile(path, bytes.Repeat([]byte{0xFF}, 4096), 0o644); err != nil {
		t.Fatalf("failed to corrupt file directly: %v", err)
	}

	// same BlockManager instance should still serve the cached value
	got, err := bm.ReadBlock(path, 0)
	if err != nil {
		t.Fatalf("ReadBlock failed: %v", err)
	}

	if !bytes.HasPrefix(got, original) {
		t.Errorf("expected cached value %q, got value reflecting disk corruption: %q", original, got[:len(original)])
	}
}

func TestReadBlock_FreshInstance_ReadsFromDiskNotStaleCache(t *testing.T) {
	path := testPath(t)

	bm1 := newTestBM(t, 4096, 10)
	if err := bm1.WriteBlock(path, 0, []byte("first writer")); err != nil {
		t.Fatalf("WriteBlock failed: %v", err)
	}

	// a fresh BlockManager (fresh cache) must read the true on-disk value
	bm2 := newTestBM(t, 4096, 10)
	got, err := bm2.ReadBlock(path, 0)
	if err != nil {
		t.Fatalf("ReadBlock failed: %v", err)
	}

	if !bytes.HasPrefix(got, []byte("first writer")) {
		t.Errorf("expected fresh instance to read disk data, got %q", got[:len("first writer")])
	}
}

func TestWriteBlock_UpdatesExistingBlockValue(t *testing.T) {
	bm := newTestBM(t, 4096, 10)
	path := testPath(t)

	if err := bm.WriteBlock(path, 0, []byte("old value")); err != nil {
		t.Fatalf("first WriteBlock failed: %v", err)
	}
	if err := bm.WriteBlock(path, 0, []byte("new value")); err != nil {
		t.Fatalf("second WriteBlock failed: %v", err)
	}

	got, err := bm.ReadBlock(path, 0)
	if err != nil {
		t.Fatalf("ReadBlock failed: %v", err)
	}

	if !bytes.HasPrefix(got, []byte("new value")) {
		t.Errorf("expected updated value, got %q", got[:len("new value")])
	}
}

func TestNew_BlockSizeNotMultipleOfPageSize_ReturnsError(t *testing.T) {
	_, err := New(5000, 10)
	if err == nil {
		t.Fatal("expected error for non-page-aligned block size, got nil")
	}
}

func TestNew_ValidBlockSize_ReturnsNoError(t *testing.T) {
	for _, size := range []int{4096, 8192, 16384} {
		if _, err := New(size, 10); err != nil {
			t.Errorf("expected no error for block size %d, got %v", size, err)
		}
	}
}

// newTestBM constructs a BlockManager for a test, failing the test
// immediately if construction returns an error.
func newTestBM(t *testing.T, blockSize int, cacheCapacity int) *BlockManager {
	t.Helper()
	bm, err := New(blockSize, cacheCapacity)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return bm
}

// testPath returns a fresh temp file path for a single test.
func testPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "blocks.bin")
}
