package wal

import "fmt"

// fakeBlockManager is an in-memory BlockManager used to test WAL logic
// before the real block manager implementation is wired in.
type fakeBlockManager struct {
	blocks map[string][]byte
}

// NewFakeBlockManager creates an empty in-memory fakeBlockManager.
func NewFakeBlockManager() *fakeBlockManager {
	return &fakeBlockManager{
		blocks: make(map[string][]byte),
	}
}

// blockKey builds the map key identifying a single block within a file.
func blockKey(path string, blockNum int) string {
	return fmt.Sprintf("%s:%d", path, blockNum)
}

// ReadBlock returns the stored block, or an error if it was never written.
func (f *fakeBlockManager) ReadBlock(path string, blockNum int) ([]byte, error) {
	data, ok := f.blocks[blockKey(path, blockNum)]
	if !ok {
		return nil, fmt.Errorf("block not found: %s block %d", path, blockNum)
	}
	return data, nil
}

// WriteBlock stores or overwrites a block in memory.
func (f *fakeBlockManager) WriteBlock(path string, blockNum int, data []byte) error {
	f.blocks[blockKey(path, blockNum)] = data
	return nil
}
