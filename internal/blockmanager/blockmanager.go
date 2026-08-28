// Package blockmanager implements the Block Manager
// using memory-mapped files for disk access.
// It is the only layer allowed to read and write files directly -
// every other component accesses disk through it.
package blockmanager

// BlockManager reads and writes fixed-size blocks to disk via mmap.
// Every block is exactly blockSize bytes; WriteBlock always writes a
// full block, zero-padding if the caller provides less data than that.
type BlockManager struct {
	blockSize int
}

// New creates a BlockManager that operates on blocks of the given size.
func New(blockSize int) *BlockManager {
	return &BlockManager{blockSize: blockSize}
}

// BlockSize returns the configured block size in bytes.
func (bm *BlockManager) BlockSize() int {
	return bm.blockSize
}
